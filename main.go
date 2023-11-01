package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	userAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.42(0x18002a2f) NetType/WIFI Language/zh_CN"
	referer   = "https://servicewechat.com/wx6c03ed6dfa30c735/595/page-frame.html"
)

var allRegions = []string{
	"江苏省", "浙江省", "福建省", "河北省", "山西省", "辽宁省", "吉林省", "黑龙江", "安徽省",
	"江西省", "山东省", "河南省", "湖北省", "湖南省", "广东省", "海南省", "四川省", "贵州省",
	"云南省", "陕西省", "甘肃省", "青海省", "内蒙古", "广西", "宁夏", "新疆", "西藏",
	"北京市", "天津市", "上海市", "重庆市", "台湾", "澳门", "香港",
}

func main() {
	all := flag.Bool("all", false, "get all regions when specified")
	details := flag.Bool("details", false, "get region details when specified")
	id := flag.String("id", "", "get only golf course with specified id (implies -details)")
	flag.Parse()

	if *id != "" {
		gc, err := getDetails(*id)
		if err != nil {
			log.Fatalln(err)
		}
		err = json.NewEncoder(os.Stdout).Encode(gc.ToGolfCourse())
		if err != nil {
			log.Fatalln(err)
		}
		return
	}
	if *all == false && flag.NArg() == 0 {
		log.Fatalln("please provide regions as arguments or use -all")
	}

	var list []golfCourseBasic
	if *all {
		list = getBasic(allRegions...)
	} else {
		list = getBasic(flag.Args()...)
	}

	os.Stdout.WriteString("[")
	for i, item := range list {
		var out interface{}
		if *details {
			gc, err := getDetails(item.ClubId)
			if err != nil {
				log.Fatalln(err)
			}
			out = gc.ToGolfCourse()
		} else {
			out = item.ToGolfCourseBasic()
		}
		b, err := json.Marshal(out)
		if err != nil {
			log.Fatalln(err)
		}
		if i > 0 {
			os.Stdout.WriteString(",")
		}
		os.Stdout.WriteString("\n")
		os.Stdout.Write(b)
	}
	os.Stdout.WriteString("\n]\n")
}

func getBasic(regions ...string) []golfCourseBasic {
	var allCourses []golfCourseBasic

	for _, region := range regions {
		log.Println("fetching", region)
		golfCourses, err := getList(region)
		if err != nil {
			log.Fatalln(err)
		}
		if len(golfCourses) < 30 {
			allCourses = append(allCourses, golfCourses...)
		} else {
			cities := getCities(region)
			if len(cities) < 1 {
				log.Fatalln(region, "needs fix")
			}
			for _, city := range cities {
				log.Println("fetching", region, city)
				golfCourses, err := getList(city)
				if err != nil {
					log.Fatalln(err)
				}
				if len(golfCourses) >= 30 {
					log.Fatalln(region, city, "needs fix")
				}
				allCourses = append(allCourses, golfCourses...)
			}
		}
	}

	sort.Slice(allCourses, func(i, j int) bool {
		id1, _ := strconv.Atoi(allCourses[i].Id)
		id2, _ := strconv.Atoi(allCourses[j].Id)
		return id1 < id2
	})

	return filterGolfCourseBasic(allCourses)
}

type golfCourseBasic struct {
	Id       string `json:"id"`
	ClubId   string `json:"club_id"`
	ClubName string `json:"club_name"`
	City     string `json:"city"`
	State    string `json:"state"`
	Address  string `json:"address"`
	Speed    string `json:"speed"`
	Lat      string `json:"lat"`
	Lng      string `json:"lng"`
}

type GolfCourseBasic struct {
	GolfLiveId int
	ClubId     string
	Name       string
	Region     string
	City       string
	Address    string
	Speed      string
	Latitude   string
	Longitude  string
}

func (gc golfCourseBasic) ToGolfCourseBasic() GolfCourseBasic {
	id, _ := strconv.Atoi(gc.Id)
	return GolfCourseBasic{
		GolfLiveId: id,
		ClubId:     gc.ClubId,
		Name:       gc.ClubName,
		Region:     gc.State,
		City:       gc.City,
		Address:    gc.Address,
		Speed:      gc.Speed,
		Latitude:   gc.Lat,
		Longitude:  gc.Lng,
	}
}

func getList(region string) ([]golfCourseBasic, error) {
	url := fmt.Sprintf("https://app.golflive.cn/index.php?s=/Home/ApiGolflive/getClubList&req_type=3&req_str=%s", region)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", referer)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error fetching data for region %s: %v\n", region, err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var response struct {
		Data []golfCourseBasic `json:"data"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("Error parsing JSON for region %s: %v\n", region, err)
	}
	return response.Data, nil
}

func filterGolfCourseBasic(courses []golfCourseBasic) (filtered []golfCourseBasic) {
	for _, course := range courses {
		if !strings.HasPrefix(course.ClubId, "UCID_") {
			filtered = append(filtered, course)
		}
	}
	return
}

type golfCourse struct {
	Club struct {
		Id         string `json:"id"`
		ClubId     string `json:"club_id"`
		ClubName   string `json:"club_name"`
		City       string `json:"city"`
		State      string `json:"state"`
		Address    string `json:"address"`
		Speed      string `json:"speed"`
		Latitude   string `json:"latitude"`
		Longitude  string `json:"longitude"`
		Phone      string `json:"phone"`
		Website    string `json:"website"`
		TotalHoles string `json:"number_of_holes"`
	} `json:"club"`
	Half []map[string]string `json:"half"`
}

type GolfCourse struct {
	GolfLiveId int
	ClubId     string
	Name       string
	Region     string
	City       string
	Address    string
	Speed      string
	Latitude   string
	Longitude  string
	Phone      string
	Website    string
	TotalHoles int
	Halves     []Half
}

type Half struct {
	Name     string
	HolePars []int
	HoleHDCP []string
}

func (gc golfCourse) ToGolfCourse() GolfCourse {
	holes, _ := strconv.Atoi(gc.Club.TotalHoles)
	halves := make([]Half, len(gc.Half))
	for i := range gc.Half {
		halves[i].Name = gc.Half[i]["half_name"]
		if halves[i].Name == "" {
			halves[i].Name = gc.Half[i]["half_id"]
		}
		var totalHoles int
		for key := range gc.Half[i] {
			if strings.HasPrefix(key, "hole") {
				n, _ := strconv.Atoi(strings.TrimPrefix(key, "hole"))
				if n > totalHoles {
					totalHoles = n
				}
			}
		}
		halves[i].HolePars = make([]int, totalHoles)
		for j := 0; j < totalHoles; j++ {
			halves[i].HolePars[j], _ = strconv.Atoi(gc.Half[i][fmt.Sprintf("hole%d", j+1)])
		}
		halves[i].HoleHDCP = make([]string, totalHoles)
		for j := 0; j < totalHoles; j++ {
			halves[i].HoleHDCP[j] = gc.Half[i][fmt.Sprintf("hdcp%d", j+1)]
		}
	}
	id, _ := strconv.Atoi(gc.Club.Id)
	return GolfCourse{
		GolfLiveId: id,
		ClubId:     gc.Club.ClubId,
		Name:       gc.Club.ClubName,
		Region:     gc.Club.State,
		City:       gc.Club.City,
		Address:    gc.Club.Address,
		Speed:      gc.Club.Speed,
		Latitude:   gc.Club.Latitude,
		Longitude:  gc.Club.Longitude,
		Phone:      gc.Club.Phone,
		Website:    gc.Club.Website,
		TotalHoles: holes,
		Halves:     halves,
	}
}

func getDetails(id string) (*golfCourse, error) {
	v := url.Values{}
	v.Set("club_id", id)
	url := "https://app.golflive.cn/index.php?s=/Home/ApiGolflive5/club_get"
	req, err := http.NewRequest("POST", url, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", referer)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error fetching details for id %s: %v\n", id, err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var response struct {
		Data golfCourse `json:"data"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("Error parsing JSON for id %s: %v\n", id, err)
	}
	return &response.Data, nil
}
