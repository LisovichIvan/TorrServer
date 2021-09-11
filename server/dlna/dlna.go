package dlna

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/anacrolix/dms/dlna/dms"
	"github.com/anacrolix/dms/upnpav"

	"server/log"
	"server/torr"
	"server/web/pages/template"
)

var dmsServer *dms.Server

func Start() {
	dmsServer = &dms.Server{
		Interfaces: func() (ifs []net.Interface) {
			var err error
			ifs, err = net.Interfaces()
			if err != nil {
				log.TLogln(err)
				os.Exit(1)
			}
			return
		}(),
		HTTPConn: func() net.Listener {
			conn, err := net.Listen("tcp", ":9080")
			if err != nil {
				log.TLogln(err)
				os.Exit(1)
			}
			return conn
		}(),
		FriendlyName:        getDefaultFriendlyName(),
		NoTranscode:         true,
		NoProbe:             true,
		StallEventSubscribe: true,
		Icons: []dms.Icon{
			//			dms.Icon{
			//				Width:      48,
			//				Height:     48,
			//				Depth:      24,
			//				Mimetype:   "image/jpeg",
			//				ReadSeeker: bytes.NewReader(template.Dlnaicon48jpg),
			//			},
			//			dms.Icon{
			//				Width:      120,
			//				Height:     120,
			//				Depth:      24,
			//				Mimetype:   "image/jpeg",
			//				ReadSeeker: bytes.NewReader(template.Dlnaicon120jpg),
			//			},
			dms.Icon{
				Width:      48,
				Height:     48,
				Depth:      24,
				Mimetype:   "image/png",
				ReadSeeker: bytes.NewReader(template.Dlnaicon48png),
			},
			dms.Icon{
				Width:      120,
				Height:     120,
				Depth:      24,
				Mimetype:   "image/png",
				ReadSeeker: bytes.NewReader(template.Dlnaicon120png),
			},
		},
		NotifyInterval: 30 * time.Second,
		AllowedIpNets: func() []*net.IPNet {
			var nets []*net.IPNet
			_, ipnet, _ := net.ParseCIDR("0.0.0.0/0")
			nets = append(nets, ipnet)
			_, ipnet, _ = net.ParseCIDR("::/0")
			nets = append(nets, ipnet)
			return nets
		}(),
		OnBrowseDirectChildren: onBrowse,
		OnBrowseMetadata:       onBrowseMeta,
	}

	if err := dmsServer.Init(); err != nil {
		log.TLogln("error initing dms server: %v", err)
		os.Exit(1)
	}
	go func() {
		if err := dmsServer.Run(); err != nil {
			log.TLogln(err)
			os.Exit(1)
		}
	}()
}

func Stop() {
	if dmsServer != nil {
		dmsServer.Close()
		dmsServer = nil
	}
}

func onBrowse(path, rootObjectPath, host, userAgent string) (ret []interface{}, err error) {
	if path == "/" {
		ret = getRoot()
		return
	} else if path == "/TR" {
		ret = getTorrents()
		return
	} else if isHashPath(path) {
		ret = getTorrent(path, host)
		return
	} else if filepath.Base(path) == "LD" {
		ret = loadTorrent(path, host)
	}
	return
}

func onBrowseMeta(path string, rootObjectPath string, host, userAgent string) (ret interface{}, err error) {
	if path == "/" {
		rootObj := upnpav.Object{
			ID:         "0",
			ParentID:   "-1",
			Restricted: 1,
			Searchable: 1,
			Title:      "TorrServer",
			Date:       upnpav.Timestamp{Time: time.Now()},
			Class:      "object.container.storageFolder",
		}
		// add Root Object
		ret = upnpav.Container{Object: rootObj, ChildCount: 1}
		return
	} else if path == "/TR" {
		// Torrents Object Meta
		trObj := upnpav.Object{
			ID:         "%2FR",
			ParentID:   "0",
			Restricted: 1,
			Searchable: 1,
			Title:      "Torrents",
			Date:       upnpav.Timestamp{Time: time.Now()},
			Class:      "object.container.storageFolder",
		}
		vol := len(torr.ListTorrent())
		ret = upnpav.Container{Object: trObj, ChildCount: vol}
		return
	} else if isHashPath(path) {
		ret = getTorrentMeta(path, host)
		return
	}
	// err = fmt.Errorf("not implemented")
	return
}

func getDefaultFriendlyName() string {
	ret := "TorrServer"
	userName := ""
	user, err := user.Current()
	if err != nil {
		log.TLogln("getDefaultFriendlyName could not get username: %s", err)
	} else {
		userName = user.Name
	}
	host, err := os.Hostname()
	if err != nil {
		log.TLogln("getDefaultFriendlyName could not get hostname: %s", err)
	}

	if userName == "" && host == "" {
		return ret
	}

	if userName != "" && host != "" {
		if userName == host {
			return ret + ": " + userName
		}
		return ret + ": " + userName + " on " + host
	}

	if host == "localhost" { // useless host, use 1st IP
		ifaces, err := net.Interfaces()
		if err != nil {
			return ret + ": " + userName + "@" + host
		}
		var list []string
		for _, i := range ifaces {
			addrs, _ := i.Addrs()
			if i.Flags&net.FlagUp == net.FlagUp {
				for _, addr := range addrs {
					var ip net.IP
					switch v := addr.(type) {
					case *net.IPNet:
						ip = v.IP
					case *net.IPAddr:
						ip = v.IP
					}
					if !ip.IsLoopback() {
						list = append(list, ip.String())
					}
				}
			}
		}
		if len(list) > 0 {
			return ret + " " + list[0]
		}
	}
	return ret + ": " + userName + "@" + host
}
