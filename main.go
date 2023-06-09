package main

import (
	"embed"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

const (
	ProgramName = "shvss"
	Version     = "v0.1.3"
	License     = `shvss
    Copyright (C) 2023 mericapewpew

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.`
	RumbleXmlUrl  = "http://rssgen.xyz/rumble/"
	OdyseeXmlUrl  = "https://odysee.com/$/rss/@"
	YoutubeXmlUrl = "https://www.youtube.com/feeds/videos.xml?channel_id="
)

var (
	//go:embed images
	images embed.FS
	//go:embed root.html
	rootHtml string
)

type SubsFile struct {
	Subs []struct {
		Name    string
		UID     string
		Service string
	}
}

type Server struct {
	wg       sync.WaitGroup
	Port     string
	SubsFile string
}

// Odysee : Odysee xml structure from https://odysee.com/$/rss/@{channelName:claimID}
type Odysee struct {
	XMLName xml.Name `xml:"rss"`
	Text    string   `xml:",chardata"`
	Dc      string   `xml:"dc,attr"`
	Content string   `xml:"content,attr"`
	Atom    string   `xml:"atom,attr"`
	Version string   `xml:"version,attr"`
	Itunes  string   `xml:"itunes,attr"`
	Channel struct {
		Text        string `xml:",chardata"`
		Title       string `xml:"title"`
		Description string `xml:"description"`
		Link        struct {
			Text string `xml:",chardata"`
			Href string `xml:"href,attr"`
			Rel  string `xml:"rel,attr"`
			Type string `xml:"type,attr"`
		} `xml:"link"`
		Image struct {
			Text  string `xml:",chardata"`
			Href  string `xml:"href,attr"`
			URL   string `xml:"url"`
			Title string `xml:"title"`
			Link  string `xml:"link"`
		} `xml:"image"`
		Generator     string `xml:"generator"`
		LastBuildDate string `xml:"lastBuildDate"`
		Language      string `xml:"language"`
		Author        string `xml:"author"`
		Category      struct {
			Text     string `xml:",chardata"`
			AttrText string `xml:"text,attr"`
		} `xml:"category"`
		Owner struct {
			Text  string `xml:",chardata"`
			Name  string `xml:"name"`
			Email string `xml:"email"`
		} `xml:"owner"`
		Explicit string `xml:"explicit"`
		Item     []struct {
			Text        string `xml:",chardata"`
			Title       string `xml:"title"`
			Description string `xml:"description"`
			Link        string `xml:"link"`
			Guid        struct {
				Text        string `xml:",chardata"`
				IsPermaLink string `xml:"isPermaLink,attr"`
			} `xml:"guid"`
			PubDate   string `xml:"pubDate"`
			Enclosure struct {
				Text   string `xml:",chardata"`
				URL    string `xml:"url,attr"`
				Length string `xml:"length,attr"`
				Type   string `xml:"type,attr"`
			} `xml:"enclosure"`
			Author string `xml:"author"`
			Image  struct {
				Text string `xml:",chardata"`
				Href string `xml:"href,attr"`
			} `xml:"image"`
			Duration string `xml:"duration"`
			Explicit string `xml:"explicit"`
		} `xml:"item"`
	} `xml:"channel"`
}

// Rumble : rumble xml structure from http://rssgen.xyz/rumble/{channelName}
type Rumble struct {
	XMLName xml.Name `xml:"rss"`
	Text    string   `xml:",chardata"`
	Version string   `xml:"version,attr"`
	Atom    string   `xml:"atom,attr"`
	Channel struct {
		Text string `xml:",chardata"`
		Link struct {
			Text string `xml:",chardata"`
			Rel  string `xml:"rel,attr"`
			Type string `xml:"type,attr"`
		} `xml:"link"`
		Title       string `xml:"title"`
		Description string `xml:"description"`
		Language    string `xml:"language"`
		Item        []struct {
			Text    string `xml:",chardata"`
			Title   string `xml:"title"`
			PubDate string `xml:"pubDate"`
			Guid    struct {
				Text        string `xml:",chardata"`
				IsPermaLink string `xml:"isPermaLink,attr"`
			} `xml:"guid"`
			Description string `xml:"description"`
			Image       struct {
				Text string `xml:",chardata"`
				Href string `xml:"href,attr"`
			} `xml:"image"`
			Thumbnail struct {
				Text string `xml:",chardata"`
				URL  string `xml:"url,attr"`
			} `xml:"thumbnail"`
		} `xml:"item"`
	} `xml:"channel"`
}

// YouTube : YouTube xml structure from https://www.youtube.com/feeds/videos.xml?channel_id={UID}
type YouTube struct {
	XMLName xml.Name `xml:"feed"`
	Text    string   `xml:",chardata"`
	Yt      string   `xml:"yt,attr"`
	Media   string   `xml:"media,attr"`
	Xmlns   string   `xml:"xmlns,attr"`
	Link    []struct {
		Text string `xml:",chardata"`
		Rel  string `xml:"rel,attr"`
		Href string `xml:"href,attr"`
	} `xml:"link"`
	ID        string `xml:"id"`
	ChannelId string `xml:"channelId"`
	Title     string `xml:"title"`
	Author    struct {
		Text string `xml:",chardata"`
		Name string `xml:"name"`
		URI  string `xml:"uri"`
	} `xml:"author"`
	Published string `xml:"published"`
	Entry     []struct {
		Text      string `xml:",chardata"`
		ID        string `xml:"id"`
		VideoId   string `xml:"videoId"`
		ChannelId string `xml:"channelId"`
		Title     string `xml:"title"`
		Link      struct {
			Text string `xml:",chardata"`
			Rel  string `xml:"rel,attr"`
			Href string `xml:"href,attr"`
		} `xml:"link"`
		Author struct {
			Text string `xml:",chardata"`
			Name string `xml:"name"`
			URI  string `xml:"uri"`
		} `xml:"author"`
		Published string `xml:"published"`
		Updated   string `xml:"updated"`
		Group     struct {
			Text    string `xml:",chardata"`
			Title   string `xml:"title"`
			Content struct {
				Text   string `xml:",chardata"`
				URL    string `xml:"url,attr"`
				Type   string `xml:"type,attr"`
				Width  string `xml:"width,attr"`
				Height string `xml:"height,attr"`
			} `xml:"content"`
			Thumbnail struct {
				Text   string `xml:",chardata"`
				URL    string `xml:"url,attr"`
				Width  string `xml:"width,attr"`
				Height string `xml:"height,attr"`
			} `xml:"thumbnail"`
			Description string `xml:"description"`
			Community   struct {
				Text       string `xml:",chardata"`
				StarRating struct {
					Text    string `xml:",chardata"`
					Count   string `xml:"count,attr"`
					Average string `xml:"average,attr"`
					Min     string `xml:"min,attr"`
					Max     string `xml:"max,attr"`
				} `xml:"starRating"`
				Statistics struct {
					Text  string `xml:",chardata"`
					Views string `xml:"views,attr"`
				} `xml:"statistics"`
			} `xml:"community"`
		} `xml:"group"`
	} `xml:"entry"`
}

type Response struct {
	Entries []Entry
}

type Entry struct {
	Service  string
	Date     string
	VidName  string
	UserName string
	VidID    string
	VidImg   string
}

func httpGet(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ERROR::http.Get('%s')::%v", url, err)
	}
	bb, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll(res.Body)::%v", err)
	}
	return bb, nil
}

func youtubeXmlUnmarshal(input []byte) (YouTube, error) {
	f := YouTube{}
	if err := xml.Unmarshal(input, &f); err != nil {
		fmt.Println(string(input))
		return YouTube{}, err
	}
	return f, nil
}

func (s *Server) writeSubsFile(sf SubsFile) error {
	jd, err := json.MarshalIndent(sf, "", " ")
	if err != nil {
		fmt.Printf("ERROR::json.MarshalIndent(sf)::%s\n", err)
	}
	if err := os.WriteFile(s.SubsFile, jd, 0666); err != nil {
		return err
	}
	return nil
}

func (s *Server) getUserName(uid string) (name string, err error) {
	if len(uid) == 24 && uid[0:2] == "UC" {
		gr, err := httpGet(YoutubeXmlUrl + uid)
		if err != nil {
			return "", fmt.Errorf("ERROR::httpGet(YoutubeXmlUrl + %s)::%v", uid, err)
		}
		f, err := youtubeXmlUnmarshal(gr)
		if err != nil {
			return "", fmt.Errorf("ERROR::youtubeXmlUnmarshal(gr)::%v", err)
		}
		return f.Author.Name, nil
	} else {
		return "", fmt.Errorf("ERROR::getUserName('%s')::NON-UUID::length=(%d) ", uid, len(uid))
	}
}

// rumbleEmbedLookup(url string) (string, err) :: GET video url and use regex to extract embed url
func rumbleEmbedLookup(url string) (string, error) {
	bb, err := httpGet(url)
	if err != nil {
		return "", fmt.Errorf("httpGet(url) :: %v\n", err)
	}
	for _, v := range strings.Split(string(bb), "\n") {
		if strings.Contains(v, "embedUrl") {
			re := regexp.MustCompile("(?i)https://rumble\\.com/embed/[a-zA-Z0-9]+/")
			return re.FindStringSubmatch(v)[0], nil
		}
	}
	return "", fmt.Errorf("failed to find 'embedUrl' in document %s", url)
}

func (s *Server) getServiceData() (Response, error) {
	r := Response{}
	fb, err := os.ReadFile(s.SubsFile)
	if err != nil {
		log.Printf("ERROR::ioutil.ReadFile(s.SubsFile)::%v\n", err)
	}
	sf := SubsFile{}
	if err := json.Unmarshal(fb, &sf); err != nil {
		return Response{}, err
	}
	for _, vv := range sf.Subs {
		s.wg.Add(1)
		vs := vv
		go func() {
			switch vs.Service {
			case "rumble":
				gr, err := httpGet(RumbleXmlUrl + vs.UID)
				if err != nil {
					log.Printf("httpGet(RumbleXmlUrl + %s)::%v\n", vs.UID, err)
					s.wg.Done()
					return
				}
				f := Rumble{}
				if err := xml.Unmarshal(gr, &f); err != nil {
					log.Printf("failed to unmarshal xml data for %s %s %s\n%v\n", vs.Service, vs.Name, vs.UID, err)
					s.wg.Done()
					return
				}
				for _, v := range f.Channel.Item {
					i := Entry{
						Service:  vs.Service,
						Date:     v.PubDate,
						VidName:  v.Title,
						UserName: vs.Name,
						VidID:    v.Guid.Text,
						VidImg:   v.Image.Href,
					}
					r.Entries = append(r.Entries, i)
				}
				s.wg.Done()
				return
			case "odysee":
				gr, err := httpGet(OdyseeXmlUrl + vs.UID)
				if err != nil {
					log.Printf("ERROR::httpGet(YOUTUBE_XML_API_ID + vs.UID)::%v\n", err)
					s.wg.Done()
					return
				}
				f := Odysee{}
				if err := xml.Unmarshal(gr, &f); err != nil {
					log.Printf("faild to unmarshal xml data for %s %s %s :: \nError=\n%v\n", vs.Service, vs.Name, vs.UID, err)
					s.wg.Done()
					return
				}
				for _, s := range f.Channel.Item {
					i := Entry{
						Service:  vs.Service,
						Date:     s.PubDate,
						VidName:  s.Title,
						UserName: s.Author,
						VidID: func() string {
							ss := strings.Split(s.Link, "/")
							return fmt.Sprintf("https://odysee.com/$/embed/@%s/%s", vs.UID, ss[len(ss)-1])
						}(),
						VidImg: s.Image.Href,
					}
					r.Entries = append(r.Entries, i)
				}
				s.wg.Done()
				return
			case "youtube":
				gr, err := httpGet(YoutubeXmlUrl + vs.UID)
				if err != nil {
					log.Printf("ERROR::httpGet(YOUTUBE_XML_API_ID + vs.UID)::%v\n", err)
					s.wg.Done()
					return
				}
				f, err := youtubeXmlUnmarshal(gr)
				if err != nil {
					log.Printf("faild to unmarshal xml data for %s %s %s :: \nError=\n%v\n", vs.Service, vs.Name, vs.UID, err)
					s.wg.Done()
					return
				}
				for _, s := range f.Entry {
					i := Entry{
						Service:  vs.Service,
						Date:     s.Published,
						VidName:  s.Title,
						UserName: s.Author.Name,
						VidID:    s.VideoId,
						VidImg:   s.Group.Thumbnail.URL,
					}
					r.Entries = append(r.Entries, i)
				}
				s.wg.Done()
				return
			}
		}()
	}
	s.wg.Wait()
	return r, nil
}

func (s *Server) subsFile(action, data, service string) (SubsFile, error) {
	sf := SubsFile{}
	fb, err := os.ReadFile(s.SubsFile)
	if err != nil {
		return sf, fmt.Errorf("ERROR::os.Open(s.SubsFile)::%s\n", err)
	}
	if err := json.Unmarshal(fb, &sf); err != nil {
		return SubsFile{}, err
	}
	switch action {
	case "remove":
		var newSubs []struct {
			Name    string
			UID     string
			Service string
		}
		if err != nil {
			return SubsFile{}, fmt.Errorf("ERROR::s.getUserName(data)::%v", err)
		}
		for _, v := range sf.Subs {
			if v.UID == data {
				continue
			}
			newSubs = append(newSubs, v)
		}
		sf.Subs = newSubs
		if err := s.writeSubsFile(sf); err != nil {
			return SubsFile{}, err
		}
		return sf, nil
	case "list":
		return sf, nil
	case "add":
		sub := struct {
			Name    string
			UID     string
			Service string
		}{}
		switch service {
		case "youtube":
			sub.Name, err = s.getUserName(data)
		case "rumble":
			sub.Name = data
		case "odysee":
			sub.Name = strings.Split(data, ":")[0]
		}
		if err != nil {
			return SubsFile{}, fmt.Errorf("ERROR::s.getUserName(data)::%v", err)
		}
		for _, v := range sf.Subs {
			if v == sub {
				//fmt.Println("sub exists")
				return sf, nil
			}
		}
		sub.UID = data
		sub.Service = service
		sf.Subs = append(sf.Subs, sub)
		log.Printf("added sub to subfile\n%v\n", sub)
		if err := s.writeSubsFile(sf); err != nil {
			return SubsFile{}, err
		}
		return sf, nil
	}
	return sf, nil
}

func (s *Server) Serve() {
	addrs, _ := net.InterfaceAddrs()
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				log.Printf("sshvss serving @ http://%v:%s\n", ipnet.IP.String(), s.Port)
				break
			}
		}
	}
	http.Handle("/images/", http.FileServer(http.FS(images)))
	http.HandleFunc("/rumbleEmbed", func(w http.ResponseWriter, r *http.Request) {
		rmu, err := rumbleEmbedLookup(r.FormValue("data"))
		if err != nil {
			log.Printf("rumbleEmbedLookup():%v\n", err)
			return
		}
		if _, err := w.Write([]byte(rmu)); err != nil {
			log.Printf("failed to write to responseWriter:%v\n", err)
			return
		}
	})
	http.HandleFunc("/subsFile", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, s.SubsFile)
	})
	http.HandleFunc("/subs", func(w http.ResponseWriter, r *http.Request) {
		bb, _ := io.ReadAll(r.Body)
		irs := struct {
			Action  string
			Value   string
			Service string
		}{}
		err := json.Unmarshal(bb, &irs)
		if err != nil {
			log.Printf("ERROR::json.Unmarshal(bb, &irs)::%v\n", err)
		}
		sf, err := s.subsFile(irs.Action, irs.Value, irs.Service)
		if err != nil {
			fmt.Printf("ERROR::s.subsFile(list)::%s\n", err)
		}
		jb, err := json.MarshalIndent(sf, "", " ")
		if err != nil {
			fmt.Printf("ERROR::json.MarshalIndent(sf)::%s\n", err)
		}
		if _, err := w.Write(jb); err != nil {
			fmt.Printf("ERROR::w.Write(jb)::%s\n", err)
		}
		return
	})
	http.HandleFunc("/videos", func(w http.ResponseWriter, r *http.Request) {
		res, err := s.getServiceData()
		if err != nil {
			if _, err := w.Write([]byte(fmt.Sprintf("ERROR::s.getServiceData()::%v\n", err))); err != nil {
				return
			}
			return
		}
		j, err := json.MarshalIndent(res, "", " ")
		if err != nil {
			_, err := w.Write([]byte(fmt.Sprintf("ERROR::json.MarshalIndent(r)::%v\n", err)))
			if err != nil {
				return
			}
			return
		}
		if _, err := w.Write(j); err != nil {
			return
		}
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t := template.New("t3")
		if _, err := t.Parse(rootHtml); err != nil {
			log.Printf("ERROR::t.Parse(rootHtml)::%v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := t.Execute(w, &struct {
			Version string
			License string
		}{
			Version: Version,
			License: License,
		}); err != nil {
			log.Printf("ERROR::t.Execute(w, &ts)::%v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	if err := http.ListenAndServe(":"+s.Port, nil); err != nil {
		fmt.Printf("ERROR::http.ListenAndServe::%v\n", err)
	}
}

func main() {
	s := Server{}
	flag.StringVar(&s.Port, "p", "8000", "Server Port")
	flag.StringVar(&s.SubsFile, "s", "subs.json", "json formatted subs file")
	b := flag.Bool("v", false, "Print version and exit")
	flag.Parse()
	if *b {
		fmt.Printf("%s %s\n", ProgramName, Version)
		return
	}
	if _, err := os.Stat(s.SubsFile); os.IsNotExist(err) {
		if f, err := os.Create(s.SubsFile); err != nil {
			return
		} else {
			if err := f.Chmod(0666); err != nil {
				return
			}
			sf := SubsFile{}
			_ = s.writeSubsFile(sf)
		}
	}
	s.Serve()
}
