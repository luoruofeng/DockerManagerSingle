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
	"log"
	"net/http"
	"strconv"
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

// 通过镜像名称，（长连接）获取该镜像的拉取进度，镜像拉取完后
func getPullImageLog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t := vars["image_id"]
	fmt.Println(t)
}

// 获取所有本地镜像
func getImages(w http.ResponseWriter, r *http.Request) {
	is, err := cm.GetAllImage()
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.ErrMes, types.EmptyOjb{})
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
		types.WriteJsonResponse(w, err, types.ErrCode, types.ErrMes, types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, i)
	}
}

// 6.通过dockerfile 构建镜像
func buildImageByDockerfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t := vars["title"]
	fmt.Println(t)
}

// 获取所有容器实例
func getContainers(w http.ResponseWriter, r *http.Request) {
	cs, err := cm.GetAllContainer()
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.ErrMes, types.EmptyOjb{})
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
		types.WriteJsonResponse(w, err, types.ErrCode, types.ErrMes, types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, c)
	}
}

// 获取容器日志根据容器id
func getContainerLog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t := vars["container_id"]
	fmt.Println(t)
}

// 镜像id，删除镜像
func deleteImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t := vars["image_id"]
	fmt.Println(t)
}

// 网络创建
func createNewwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t := vars["image_id"]
	fmt.Println(t)
}

// 根据id获取网络信息
func getNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nid := vars["network_id"]
	n, err := cm.GetNetworkById(nid)
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.ErrMes, types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, n)
	}
}

// 获取所有网络信息
func getNetworks(w http.ResponseWriter, r *http.Request) {
	ns, err := cm.GetAllNetwork()
	if err != nil {
		types.WriteJsonResponse(w, err, types.ErrCode, types.ErrMes, types.EmptyOjb{})
	} else {
		types.WriteJsonResponse(w, err, types.SucCode, types.SucMes, ns)
	}
}

// 删除网络根据网络id
func deleteNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t := vars["network_id"]
	fmt.Println(t)
}

// 通过镜像名称+版本号+CPU+CMD命令+Privileged(boolean)+内存+硬盘数+进程数+环境变量map+是否暴露端口，启动对应的容器，并启动，返回容器的所有信息
func containerCreateAndRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t := vars["network_id"]
	fmt.Println(t)
}

// 启动对应的容器
func containerRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t := vars["container_id"]
	fmt.Println(t)
}

// 容器id，停止，删除容器
func containerStop(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t := vars["container_id"]
	fmt.Println(t)
}

// 容器id，停止，删除容器
func containerStopAndRemove(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t := vars["network_id"]
	fmt.Println(t)
}

// 通过容器id，进入容器，(长连接)并可以通过命令操作容器
func operateContainer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	t := vars["network_id"]
	fmt.Println(t)
}

func addHandler(r *mux.Router) {
	r.HandleFunc("/image", downloadImage).Methods("POST")
	r.HandleFunc("/image/log/{image_id}", getPullImageLog).Methods("GET")
	r.HandleFunc("/images", getImages).Methods("GET")
	r.HandleFunc("/image/{image_id}", getImage).Methods("GET")
	r.HandleFunc("/image/{image_id}", deleteImage).Methods("DELETE")
	r.HandleFunc("/image/build", buildImageByDockerfile).Methods("POST")
	r.HandleFunc("/containers", getContainers).Methods("GET")
	r.HandleFunc("/container/{container_id}", getContainerInfo).Methods("GET")
	r.HandleFunc("/container/log/{container_id}", getContainerLog).Methods("GET")
	r.HandleFunc("/container", containerCreateAndRun).Methods("POST")
	r.HandleFunc("/container/start/{container_id}", containerRun).Methods("PUT")
	r.HandleFunc("/container/stop/{container_id}", containerStop).Methods("PUT")
	r.HandleFunc("/container/{container_id}", containerStopAndRemove).Methods("DELETE")
	r.HandleFunc("/network", createNewwork).Methods("POST")
	r.HandleFunc("/network/{network_id}", getNetwork).Methods("GET")
	r.HandleFunc("/networks", getNetworks).Methods("GET")
	r.HandleFunc("/network/{network_id}", deleteNetwork).Methods("DELETE")
	r.HandleFunc("/container/operation/{container_id}", operateContainer).Methods("POST")
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
		log.Printf("api is running... port:%d \n", port)
		err := srv.ListenAndServe()
		if err != nil {
			log.Println("web_api server is stoped. " + err.Error())
			return err
		} else {
			return nil
		}
	})

	e.Go(func() error {
		<-ctx.Done()
		log.Println("web_api is shutting down now")
		return srv.Shutdown(context.Background())
	})
}
