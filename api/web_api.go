// Package classification User API.
//
// The purpose of this service is to provide an application
// that is using plain go code to define an API
//
//      Host: localhost
//      Version: 0.0.1
//
// swagger:meta
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/luoruofeng/dockermanagersingle/container"

	"github.com/gorilla/mux"
	"github.com/luoruofeng/dockermanagersingle/types"
	"golang.org/x/sync/errgroup"
)

var cm container.ContainerManager

// 传镜像名称+版本号，下载镜像
func downloadImage(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /image image getImage
	//
	// get image by image_name:image_version
	//
	// This will pull docker image
	//
	//     Responses:
	//       200: types.ImageInfo
	vars := mux.Vars(r)
	t := vars["title"]
	fmt.Println(t)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(types.ImageInfo{})
}

// 获取所有本地镜像
func getImages(w http.ResponseWriter, r *http.Request) {
	is, err := cm.GetAllImage()
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, is)
	}
}

// 获取本地镜像
func getImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	iid := vars["image_id"]
	i, err := cm.GetImageById(iid)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, i)
	}
}

// 6.通过dockerfile 构建镜像
func buildImageByDockerfile(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	f, fh, err := r.FormFile("dockerfile")
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.ErrParamMes, types.EmptyOjb{})
		return
	}
	defer f.Close()

	log.Printf("build image by dockerfile.file head:%v\n", fh)
	b := make([]byte, 1024)

	for {
		n, err := f.Read(b)
		if err != nil {
			if n <= 0 && err == io.EOF {
				break
			} else {
				types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
				return
			}
		}
		if n > 0 {
			b = append(b, b[:n]...)
		}
	}
	fc := string(b)
	log.Printf("build image by dockerfile. dockerfile content:%v\n", fc)
	br, err := cm.BuildImage(fc)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, br)
	}
}

// 获取所有容器实例
func getContainers(w http.ResponseWriter, r *http.Request) {
	cs, err := cm.GetAllContainer()
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, cs)
	}
}

// 获取容器信息根据容器id
func getContainerInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["container_id"]
	c, err := cm.GetContainerById(cid)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, c)
	}
}

// 获取容器日志根据容器id
func getContainerLog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["container_id"]
	rc, err := cm.GetContainerLogById(cid)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
		return
	}
	defer func() {
		err := rc.Close()
		if err != nil {
			log.Println(err)
		}
	}()
	bs, err := io.ReadAll(rc)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, string(bs))
	}
}

// 镜像id，删除镜像
func deleteImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	iid := vars["image_id"]
	err := cm.DeleteImageById(iid)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, iid)
	}
}

func connectNewwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["container_id"]
	nid := vars["network_id"]
	err := cm.ConnectNetwork(nid, cid)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, nil)
	}
}

func disconnectNewwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["container_id"]
	nid := vars["network_id"]
	err := cm.DisconnectNetwork(nid, cid)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, nil)
	}
}

// 网络创建
func createNewwork(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	subnet := r.Form.Get("subnet")
	gateway := r.Form.Get("gateway")
	name := r.Form.Get("name")
	log.Printf("create network name:%v subnet:%v gateway:%v\n", name, subnet, gateway)
	nr, err := cm.CreateNetwork(name, subnet, gateway)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, nr)
	}
}

// 根据id获取网络信息
func getNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nid := vars["network_id"]
	n, err := cm.GetNetworkById(nid)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, n)
	}
}

// 获取所有网络信息
func getNetworks(w http.ResponseWriter, r *http.Request) {
	ns, err := cm.GetAllNetwork()
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, ns)
	}
}

// 删除网络根据网络id
func deleteNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nid := vars["network_id"]
	err := cm.DeleteNetworkById(nid)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, nid)
	}
}

type ContainerCreateParam struct {
	ContainerName string            `json:"container_name"`
	ImageId       string            `json:"image_id"`
	Cmd           string            `json:"cmd"`
	Envs          map[string]string `json:"envs"`
	Ports         []int             `json:"ports"`
}

func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// 通过镜像名称+版本号+CPU+CMD命令+Privileged(boolean)+内存+硬盘数+进程数+环境变量map+是否暴露端口，启动对应的容器，并启动，返回容器的所有信息
func containerCreate(w http.ResponseWriter, r *http.Request) {

	var param ContainerCreateParam

	r.ParseForm()
	rb := r.Body
	defer rb.Close()
	if err := json.NewDecoder(rb).Decode(&param); err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
		return
	}

	var mp map[int]int
	if len(param.Ports) > 0 {
		mp = make(map[int]int)
		for _, p := range param.Ports {
			fp, err := GetFreePort()
			if err != nil {
				types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
				return
			}
			mp[fp] = p
		}
	}
	var cmd []string
	if param.Cmd != "" {
		cmd = strings.Split(param.Cmd, " ")
	}

	var envs []string
	if param.Envs != nil {
		envs = make([]string, 0)
		for k, v := range param.Envs {
			envs = append(envs, strings.ReplaceAll(k, "=", "\\=")+"="+strings.ReplaceAll(v, "=", "\\="))
		}
	}

	log.Println("create container .params is ", param)
	ccb, err := cm.CreateContainer(param.ImageId, envs, cmd, mp, param.ContainerName)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, ccb)
	}
}

// 启动对应的容器
func containerRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["container_id"]
	err := cm.StartContainer(cid)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, cid)
	}
}

// 容器id，停止，删除容器
func containerStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid := vars["container_id"]
	err := cm.StopContainerById(cid)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, cid)
	}
}

// 容器id，停止，删除容器
func containerStopAndRemove(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nid := vars["container_id"]
	err := cm.DeleteContainerById(nid)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.Mes(err.Error()), types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, nid)
	}
}

func addHandler(r *mux.Router) {
	r.HandleFunc("/images", getImages).Methods("GET")
	r.HandleFunc("/image/{image_id}", getImage).Methods("GET")
	r.HandleFunc("/image/{image_id}", deleteImage).Methods("DELETE")
	r.HandleFunc("/image/build", buildImageByDockerfile).Methods("POST")
	r.HandleFunc("/containers", getContainers).Methods("GET")
	r.HandleFunc("/container/{container_id}", getContainerInfo).Methods("GET")
	r.HandleFunc("/container/log/{container_id}", getContainerLog).Methods("GET")
	r.HandleFunc("/container", containerCreate).Methods("POST")
	r.HandleFunc("/container/start/{container_id}", containerRun).Methods("PUT")
	r.HandleFunc("/container/stop/{container_id}", containerStop).Methods("PUT")
	r.HandleFunc("/container/{container_id}", containerStopAndRemove).Methods("DELETE")
	r.HandleFunc("/network", createNewwork).Methods("POST")
	r.HandleFunc("/network/conn/{container_id}/{network_id}", connectNewwork).Methods("POST")
	r.HandleFunc("/network/disconn/{container_id}/{network_id}", disconnectNewwork).Methods("POST")
	r.HandleFunc("/network/{network_id}", getNetwork).Methods("GET")
	r.HandleFunc("/networks", getNetworks).Methods("GET")
	r.HandleFunc("/network/{network_id}", deleteNetwork).Methods("DELETE")
}

func Start(ctx context.Context, e *errgroup.Group, port int, readTimeout int, writeTimeout int, idleTimeout int) {
	cm = container.GetCM()
	r := mux.NewRouter()
	addHandler(r)

	srv := &http.Server{
		Addr: "0.0.0.0:" + strconv.Itoa(port),
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * time.Duration(writeTimeout),
		ReadTimeout:  time.Second * time.Duration(readTimeout),
		IdleTimeout:  time.Second * time.Duration(idleTimeout),
		Handler:      r, // Pass our instance of gorilla/mux in.
	}

	e.Go(func() error {
		log.Printf("api server is running... port:%d \n", port)
		err := srv.ListenAndServe()
		if err != nil {
			log.Println("web_api server is stoped. error:" + err.Error())
			return err
		} else {
			log.Println("web_api server is stoped gracefully. ")
			return nil
		}
	})

	e.Go(func() error {
		<-ctx.Done()
		log.Println("web_api is shutting down now")
		return srv.Shutdown(context.Background())
	})
}
