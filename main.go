package main

import (
	// "crypto/md5"
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/urfave/cli"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	volumes []string
	fmap    = make(map[string]string)
)

func main() {
	app := cli.NewApp()
	app.Name = "minibar"
	app.Usage = "lightweight simple distributed storage system"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "p",
			Value: "8090",
			Usage: "port to listen",
		},
		cli.StringFlag{
			Name:  "mserver",
			Usage: "master server to mount  (only work for volume server)",
		},
		cli.StringFlag{
			Name:  "dir",
			Usage: "directory to store file (only work for volume server)",
		},
		cli.StringFlag{
			Name:  "max",
			Usage: "max space use           (only work for volume server)",
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.Args().First() == "master" {
			return masterServer(c)
		} else if c.Args().First() == "volume" {
			return volumeServer(c)
		} else {
			fmt.Println("invalid input, type ' minibar -h ' for help")
		}
		return nil
	}
	app.Run(os.Args)
}
func masterServer(c *cli.Context) error {
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			size, err := strconv.Atoi(r.Header.Get("Content-Length"))
			fmt.Println("size:", size)
			if err != nil {
				fmt.Fprint(w, err)
				fmt.Println("Get size failed", err)
				return
			}
			for _, v := range volumes {
				res, err := http.Get("http://" + v + "/getSpaceRemain")
				if err != nil {
					fmt.Fprint(w, err)
					fmt.Println(err)
					return
				}
				defer res.Body.Close()
				rbody, _ := ioutil.ReadAll(res.Body)
				lsize, _ := strconv.Atoi(string(rbody))
				if lsize < size {
					fmt.Println(v, "insufficient space , goto next one")
					continue
				} else {
					fmt.Println("==== ", v, "====")
					r.ParseMultipartForm(32 << 20)
					file, handler, err := r.FormFile("uploadFile")
					if err != nil {
						fmt.Println(err)
						return
					}
					defer file.Close()

					bodyBuf := &bytes.Buffer{}
					bodyWriter := multipart.NewWriter(bodyBuf)
					fileWriter, err := bodyWriter.CreateFormFile("uploadFile", handler.Filename)
					if err != nil {
						fmt.Println("error writing to buffer")
						fmt.Fprint(w, err)
						return
					}

					io.Copy(fileWriter, file)
					contenType := bodyWriter.FormDataContentType()
					bodyWriter.Close()
					resp, err := http.Post("http://"+v+"/upload", contenType, bodyBuf)
					if err != nil {
						fmt.Println("do upload failed", err)
						fmt.Fprint(w, err)
						return
					}
					rpbody, _ := ioutil.ReadAll(resp.Body)
					fmt.Fprint(w, string(rpbody))
					fmap[string(rpbody)] = v
					fmt.Println("stored file", string(rpbody), "into", v)
					return
				}
			}
			fmt.Fprint(w, "insufficient storage space")
		}
	})
	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		vip := fmap[r.FormValue("fid")]
		if vip == "" {
			fmt.Fprint(w, "no such file")
			return
		}
		fmt.Fprint(w, "http://"+vip+"/download?fid="+r.FormValue("fid"))
		return
	})
	http.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		volumeIP := fmap[r.FormValue("fid")]
		if volumeIP == "" {
			fmt.Fprint(w, "no such file")
			return
		}
		res, err := http.Get("http://" + volumeIP + "/delete?fid=" + r.FormValue("fid"))
		if err != nil {
			fmt.Fprint(w, err)
			return
		}
		defer res.Body.Close()
		rbody, _ := ioutil.ReadAll(res.Body)
		fmt.Fprint(w, string(rbody))
	})
	http.HandleFunc("/registerVolume", func(w http.ResponseWriter, r *http.Request) {
		volumeIP := r.FormValue("volumeIP")
		if volumeIP != "" {
			volumes = append(volumes, volumeIP)
			fmt.Println("new volume", volumeIP)
			return
		}
		fmt.Println("register volume failed")
	})
	err := http.ListenAndServe(":"+c.String("p"), nil)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func volumeServer(c *cli.Context) error {
	mdir := c.String("dir")
	if mdir == "" {
		fmt.Println("-p option is needed")
		fmt.Println("'minibar -h' for more help")
		return nil
	}
	if mdir[len(mdir)-1:] != "/" {
		mdir += "/"
	}
	max := c.String("max")
	var freeSpace int64
	freeSpace, err := strconv.ParseInt(max, 10, 64)
	if err != nil {
		freeSpace = DiskUsage(mdir)
	}
	_, err = http.Get("http://" + c.String("mserver") + "/registerVolume?volumeIP=" + getIP() + ":" + c.String("p"))
	if err != nil {
		fmt.Println("register to master server:", c.String("mserver"), "failed ,", err.Error())
		return err
	}
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseMultipartForm(32 << 20)
			file, handler, err := r.FormFile("uploadFile")
			if err != nil {
				fmt.Println(err)
				return
			}
			defer file.Close()
			crutime := time.Now().Unix()
			h := md5.New()
			io.WriteString(h, strconv.FormatInt(crutime, 10))
			token := fmt.Sprintf("%x", h.Sum(nil))
			fname := token
			if strings.Contains(handler.Filename, ".") {
				fname += handler.Filename[strings.LastIndex(handler.Filename, "."):]
			}
			f, err := os.OpenFile(mdir+fname, os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer f.Close()
			io.Copy(f, file)
			fmt.Fprint(w, fname)
		}
	})
	http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.OpenFile(mdir+r.FormValue("fid"), os.O_RDONLY, 0666)
		if err != nil {
			fmt.Fprint(w, "no such file")
			return
		}
		f.Close()
		w.Header().Add("Content-Disposition", "attachment; filename="+r.FormValue("fid"))
		w.Header().Add("Content-Type", "application/octet-stream")
		http.ServeFile(w, r, mdir+r.FormValue("fid"))
	})
	http.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		fid := r.FormValue("fid")
		f, err := os.OpenFile(mdir+fid, os.O_WRONLY, 0666)
		if err != nil {
			fmt.Fprint(w, "OK")
			return
		}
		fi, _ := f.Stat()
		freeSpace -= fi.Size()
		f.Close()
		err = os.Remove(mdir + fid)
		if err != nil {
			fmt.Println("remove file failed:", err)
			fmt.Fprint(w, "WRONG")
			return
		}
		fmt.Fprint(w, "OK")
	})
	http.HandleFunc("/getSpaceRemain", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strconv.FormatInt(freeSpace, 10))
	})
	fmt.Println("-p = ", c.String("p"))
	err = http.ListenAndServe(":"+c.String("p"), nil)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
func getIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		fmt.Println(err)
		return ""
	}
	var strs []string
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			fmt.Println(err)
			continue
		}
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ip := v.IP
				strs = append(strs, ip.String())
			case *net.IPAddr:
				// ip := v.IP
				// strs = append(strs, ip.String())
			}
		}
	}
	for _, v := range strs {
		if strings.HasPrefix(v, "192.168.") {
			return v
		}
	}
	for _, v := range strs {
		if strings.HasPrefix(v, "10.") {
			return v
		}
	}
	for _, v := range strs {
		if strings.HasPrefix(v, "172.") {
			return v
		}
	}
	for _, v := range strs {
		if v != "127.0.0.1" && v != "::1" {
			return v
		}
	}
	return strs[0]
}
func DiskUsage(path string) int64 {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return 0
	}
	return int64(fs.Bfree * uint64(fs.Bsize))
}
