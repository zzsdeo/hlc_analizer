package main

import (
	"archive/zip"
	"bufio"
	"flag"
	"fmt"
	"github.com/mailru/easyjson"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
)

type stat struct {
	accounts     map[string]int
	filter       map[string]int
	sname_starts map[string]int
}

type resultString struct {
	key string
	qty int
}

var accounts []Account

//easyjson:json
type Account struct {
	ID        int      `json:"id,omitempty"`        //unique
	Email     string   `json:"email,omitempty"`     //up to 100 symbols, unique
	FName     string   `json:"fname,omitempty"`     //up to 50 symbols, optional
	SName     string   `json:"sname,omitempty"`     //up to 50 symbols, optional
	Phone     string   `json:"phone,omitempty"`     //up to 16 symbols, unique, optional
	Sex       string   `json:"sex,omitempty"`       //m|f
	Birth     int      `json:"birth,omitempty"`     //timestamp from 01.01.1950 to 01.01.2005
	Country   string   `json:"country,omitempty"`   //up to 50 symbols, optional
	City      string   `json:"city,omitempty"`      //up to 50 symbols, optional, every city belongs to defined country
	Joined    int      `json:"joined,omitempty"`    //timestamp from 01.01.2011 to 01.01.2018
	Status    string   `json:"status,omitempty"`    //"свободны", "заняты", "всё сложно"
	Interests []string `json:"interests,omitempty"` //every string is up to 100 symbols, optional
	Premium   *Premium `json:"premium,omitempty"`
	Likes     []Like   `json:"likes,omitempty"`
}

//easyjson:json
type Premium struct {
	Start  int `json:"start,omitempty"`  //timestamp from 01.01.2018
	Finish int `json:"finish,omitempty"` //timestamp from 01.01.2018
}

//easyjson:json
type Like struct {
	ID int `json:"id,omitempty"` //id of the liked account
	TS int `json:"ts,omitempty"` //timestamp when like has been set
}

//easyjson:json
type Accounts struct {
	Accounts []Account `json:"accounts"`
}

func main() {
	ammos := flag.String("a", "", "ammo files")
	data := flag.String("d", "", "data.zip file")
	flag.Parse()

	if *data != "" {
		r, err := readZip(*data)
		if err != nil {
			log.Fatal("[ERROR] ", err)
		}
		for _, file := range r.File {
			data, err := parseData(file)
			if err != nil {
				log.Fatal("[ERROR] ", err)
			}
			accounts = append(accounts, data...)
		}
		err = r.Close()
		if err != nil {
			log.Fatal("[ERROR] ", err)
		}
	}

	s := stat{
		accounts:     map[string]int{"id": len(accounts), "sex": 2, "email": len(accounts), "birth": len(accounts), "joined": len(accounts), "status": 3, "premium": 3},
		filter:       map[string]int{},
		sname_starts: map[string]int{},
	}

	for _, account := range accounts {
		if account.FName != "" {
			if _, ok := s.accounts["fname"]; !ok {
				s.accounts["fname"] = 0
			}
			s.accounts["fname"]++
		}
		if account.SName != "" {
			if _, ok := s.accounts["sname"]; !ok {
				s.accounts["sname"] = 0
			}
			s.accounts["sname"]++
		}
		if account.Phone != "" {
			if _, ok := s.accounts["phone"]; !ok {
				s.accounts["phone"] = 0
			}
			s.accounts["phone"]++
		}
		if account.Country != "" {
			if _, ok := s.accounts["country"]; !ok {
				s.accounts["country"] = 0
			}
			s.accounts["country"]++
		}
		if account.City != "" {
			if _, ok := s.accounts["city"]; !ok {
				s.accounts["city"] = 0
			}
			s.accounts["city"]++
		}
		if account.Interests != nil {
			if _, ok := s.accounts["interests"]; !ok {
				s.accounts["interests"] = 0
			}
			s.accounts["interests"] += len(account.Interests)
		}
		if account.Likes != nil {
			if _, ok := s.accounts["likes"]; !ok {
				s.accounts["likes"] = 0
			}
			s.accounts["likes"] += len(account.Likes)
		}
	}

	for _, ammo := range strings.Split(*ammos, " ") {
		err := readFile(ammo, s)
		if err != nil {
			log.Println(err)
		}
	}

	for k, v := range s.sname_starts {
		fmt.Println(k, v)
	}

	//fmt.Println("by qty -----------------------------------------")
	//for _, v := range s.exposeAmmo(false) {
	//	fmt.Println(v.key, v.qty)
	//}
	//
	//fmt.Println("by key -----------------------------------------")
	//for _, v := range s.exposeAmmo(true) {
	//	fmt.Println(v.key, v.qty)
	//}
	//
	//fmt.Println("accounts -----------------------------------------")
	//for _, v := range s.exposeAccounts() {
	//	fmt.Println(v.key, v.qty)
	//}
}

func readFile(path string, s stat) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s.add(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (s *stat) add(line string) {
	if strings.Contains(line, "GET /accounts/filter/?") {
		line = strings.TrimLeft(line, "GET ")
		line = strings.TrimRight(line, " HTTP/1.1")

		u, _ := url.Parse(line)
		var key string
		var keyParts []string

		for arg, v := range u.Query() {
			if strings.Contains(arg, "sex_") {
				keyParts = append(keyParts, "sex")
			} else if strings.Contains(arg, "email_") {
				keyParts = append(keyParts, "email")
			} else if strings.Contains(arg, "status_") {
				keyParts = append(keyParts, "status")
			} else if strings.Contains(arg, "fname_") {
				keyParts = append(keyParts, "fname")
			} else if strings.Contains(arg, "sname_") {
				s.sname_starts[v[0]]++
				keyParts = append(keyParts, "sname")
			} else if strings.Contains(arg, "phone_") {
				keyParts = append(keyParts, "phone")
			} else if strings.Contains(arg, "country_") {
				keyParts = append(keyParts, "country")
			} else if strings.Contains(arg, "city_") {
				keyParts = append(keyParts, "city")
			} else if strings.Contains(arg, "birth_") {
				keyParts = append(keyParts, "birth")
			} else if strings.Contains(arg, "interests_") {
				keyParts = append(keyParts, "interests")
			} else if strings.Contains(arg, "likes_") {
				keyParts = append(keyParts, "likes")
			} else if strings.Contains(arg, "premium_") {
				keyParts = append(keyParts, "premium")
			}
		}

		sort.Slice(keyParts, func(i, j int) bool {
			return s.accounts[keyParts[i]] < s.accounts[keyParts[j]]
		})

		for i, k := range keyParts {
			if i < len(keyParts)-1 {
				key += k + "+"
			} else {
				key += k
			}
		}

		if _, ok := s.filter[key]; !ok {
			s.filter[key] = 1
			return
		}

		s.filter[key]++
	}
}

func (s *stat) exposeAmmo(byKey bool) []resultString {
	var res []resultString
	for k, v := range s.filter {
		res = append(res, resultString{
			key: k,
			qty: v,
		})
	}

	if byKey {
		sort.Slice(res, func(i, j int) bool {
			return res[i].key < res[j].key
		})
	} else {
		sort.Slice(res, func(i, j int) bool {
			return res[i].qty > res[j].qty
		})
	}

	return res
}

func (s *stat) exposeAccounts() []resultString {
	var res []resultString
	for k, v := range s.accounts {
		res = append(res, resultString{
			key: k,
			qty: v,
		})
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].qty < res[j].qty
	})

	return res
}

func readZip(dataFilePath string) (*zip.ReadCloser, error) {
	r, err := zip.OpenReader(dataFilePath)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func parseData(f *zip.File) ([]Account, error) {
	accounts := Accounts{}
	file, err := f.Open()
	if err != nil {
		return accounts.Accounts, err
	}

	err = easyjson.UnmarshalFromReader(file, &accounts)
	if err != nil {
		return accounts.Accounts, err
	}

	err = file.Close()
	if err != nil {
		log.Println("[ERROR] ", err)
	}

	return accounts.Accounts, nil
}
