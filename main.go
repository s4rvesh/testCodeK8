package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/exec"
	"strconv"
)

func main(){

	fmt.Println("savvy K8s running")
	//K8sProxy()

	//Add loop to run after specific time

	var PodResponseObject PodMetrics
	var NodeResponseObject NodeMetrics


	PodResponseObject = GetPods()
	NodeResponseObject = GetNodes()
	PodResponseObject, NodeResponseObject = GetIntVals(PodResponseObject,NodeResponseObject)

	fmt.Println("-------------------------------------------------")
	fmt.Println(PodResponseObject)
	fmt.Println("")
	fmt.Println("")
	fmt.Println(NodeResponseObject)
	fmt.Println("-------------------------------------------------")

	CheckThresholdPod(PodResponseObject)
	CheckThresholdNode(NodeResponseObject)


	//mongostore.MongoStore(PodResponseObject, NodeResponseObject)
}



////////////////////////////////////////////////////////
func K8sProxy(){

	out, err := exec.Command("kubectl", "proxy", "--port=8080").Output()
	if err != nil {
		fmt.Printf("%s", err)
	}
	fmt.Println("Command Successfully Executed")
	output := string(out[:])
	fmt.Println(output)

}
////////////////////////////////////////////////////////////

func GetPods() PodMetrics{

	url := "http://127.0.0.1:8080/apis/metrics.k8s.io/v1beta1/pods"
	responseData := Getdata(url)

	var PodResponseObject PodMetrics
	json.Unmarshal(responseData, &PodResponseObject)

	fmt.Println(PodResponseObject)

	return PodResponseObject
}
////////////////////////////////////////////

func GetNodes() NodeMetrics{

	url := "http://127.0.0.1:8080/apis/metrics.k8s.io/v1beta1/nodes"
	responseData := Getdata(url)

	var NodeResponseObject NodeMetrics
	json.Unmarshal(responseData, &NodeResponseObject)

	return NodeResponseObject
}

/////////////////////////////////////////////////////////

func Getdata(url string) []byte{

	response , err :=http.Get(url)
	if err!=nil{
		fmt.Println(err.Error())
		os.Exit(1)
	}

	responseData , err := ioutil.ReadAll(response.Body)
	if err!=nil{
		log.Fatal(err)
	}
	return responseData
}
//////////////////////////////////////////////////////


func GetIntVals(PodResponseObject PodMetrics,NodeResponseObject NodeMetrics)(PodMetrics, NodeMetrics){

	for  i:=0;i<len(PodResponseObject.Pods);i++{
		j:=0
		for ;j<len(PodResponseObject.Pods[i].Containers);j++{
			k:=0
			for ;k<len(PodResponseObject.Pods[i].Containers[j].ContainerUsages);j++{
				PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].CpuInt,PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].MemoryInt = convertInt(PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].Cpu, PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].Memory)
			}
		}

	}


	for i:=0;i<len(NodeResponseObject.Nodes);i++{
		NodeResponseObject.Nodes[i].NodeUsages.CpuInt,NodeResponseObject.Nodes[i].NodeUsages.MemoryInt=convertInt(NodeResponseObject.Nodes[i].NodeUsages.Cpu,NodeResponseObject.Nodes[i].NodeUsages.Memory)
	}


	return PodResponseObject,NodeResponseObject
}



func convertInt(cpuMetrics string, memoryMetrics string) (int64,int64){
	if last := len(cpuMetrics) - 1; last >= 0 && cpuMetrics[last] == 'n' {
		cpuMetrics = cpuMetrics[:last]
	}

	if last := len(memoryMetrics) - 1; last >= 0 && memoryMetrics[last] == 'i' {
		memoryMetrics = memoryMetrics[:last]

		if last := len(memoryMetrics) - 1; last >= 0 && memoryMetrics[last] == 'K' {
			memoryMetrics = memoryMetrics[:last]
		}else if last := len(memoryMetrics) - 1; last >= 0 && memoryMetrics[last] == 'M' {
			memoryMetrics = memoryMetrics[:last]
		}else if last := len(memoryMetrics) - 1; last >= 0 && memoryMetrics[last] == 'G' {
			memoryMetrics = memoryMetrics[:last]
		}
	}

	cpuMetricsInt, _ := strconv.ParseInt(cpuMetrics,10,64)
	memoryMetricsInt,_ := strconv.ParseInt(memoryMetrics,10,64)

	return cpuMetricsInt,memoryMetricsInt

}
////////////////////////////////////////////////////////////

func CheckThresholdNode(NodeResponseObject NodeMetrics){

	for i:=0;i<len(NodeResponseObject.Nodes);i++{
		if NodeResponseObject.Nodes[i].NodeUsages.CpuInt > 10000{
			MailAlert("Node",NodeResponseObject.Nodes[i].MetadataNodes.Name,"cpu",NodeResponseObject.Nodes[i].NodeUsages.CpuInt )

		} else if NodeResponseObject.Nodes[i].NodeUsages.MemoryInt > 10000{
			MailAlert("Node",NodeResponseObject.Nodes[i].MetadataNodes.Name,"memory",NodeResponseObject.Nodes[i].NodeUsages.MemoryInt)

		}
	}

}


func CheckThresholdPod(PodResponseObject PodMetrics){


	for i:=0;i<len(PodResponseObject.Pods);i++{

		for j:=0;j<len(PodResponseObject.Pods[i].Containers);j++{

			for k:=0;k<len(PodResponseObject.Pods[i].Containers[j].ContainerUsages);k++{

				if PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].CpuInt > 1000{

					MailAlert("Pod",PodResponseObject.Pods[i].MetadataPods.Name,"cpu",PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].CpuInt )

				} else if PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].MemoryInt > 1000 {

					MailAlert("Pod",PodResponseObject.Pods[i].MetadataPods.Name,"cpu",PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].MemoryInt )

				}


			}
		}

	}


}

////////////////////////////////////////////////////////////////////



func (s *smtpServer) Address() string {
	return s.host + ":" + s.port
}

func MailAlert(item string,item_name string, metric_type string, metric_val int64){

	from := "sarvesh.upadhye@gmail.com"
	password := "S4rvesh@7781"



	// Receiver email address to be set

	to := []string{
		"hashercool@gmail.com",
	}

	smtpServer := smtpServer{host: "smtp.gmail.com", port: "587"}



	var message []byte
	if item=="node"{
		if metric_type=="memory"{
			m1:="The memory usage of Node:"
			m2:=item_name
			m3:=" is above threshold. Memory Usage:"
			m4:=metric_val
			message=[]byte(m1+m2+m3+strconv.FormatInt(m4, 10))
		} else {
			m1:="The CPU usage of Node:"
			m2:=item_name
			m3:=" is above threshold. CPU Usage:"
			m4:=metric_val
			message=[]byte(m1+m2+m3+strconv.FormatInt(m4, 10))
		}


	} else {
		if metric_type=="memory"{
			m1:="The memory usage of Pod:"
			m2:=item_name
			m3:=" is above threshold. Memory Usage:"
			m4:=metric_val
			message=[]byte(m1+m2+m3+strconv.FormatInt(m4, 10))
		} else {
			m1:="The CPU usage of Pod:"
			m2:=item_name
			m3:=" is above threshold. CPU Usage:"
			m4:=metric_val
			message=[]byte(m1+m2+m3+strconv.FormatInt(m4, 10))
		}
	}



	auth := smtp.PlainAuth("", from, password, smtpServer.host)
	err := smtp.SendMail(smtpServer.Address(), auth, from, to, message)
	if err != nil {
		fmt.Println(err)
	}


}








/////////////////////////////////////////////////////////////
type smtpServer struct {
	host string
	port string
}


type NodeMetrics struct{

	Kind string `json:"kind"`
	ApiVersion string `json: "apiVersion"`
	Metadata string `json:"metadata"`
	Nodes []Node `json:"items"`
}

type Node struct {
	MetadataNodes MetadataNode `json:"metadata"`
	Timestamp string `json:"timestamp"`
	Window string `json:"window"`
	NodeUsages NodeUsage `json:"usage"`
}

type NodeUsage struct{
	Cpu string `json:"cpu"`
	Memory string `json:"memory"`
	CpuInt int64
	MemoryInt int64
}
type  MetadataNode struct {
	Name string `json:"name"`
	SelfLink string `json:"selfLink"`
	CreationTimeStamp string `json:"creationTimestamp"`
}


type PodMetrics struct{

	Kind string `json:"kind"`
	ApiVersion string `json:"apiVersion"`
	Metadata string `json:"metadata"`
	SelfLink string `json:"selfLink"`
	Pods []Pod `json:"items"`
}


type Pod struct {
	MetadataPods MetadataPod `json:"metadata"`
	Timestamp string `json:"timestamp"`
	Window string `json:"window"`
	Containers []Container `json:"containers"`
}



type Container struct{
	Name string `json:"name"`
	ContainerUsages []ContainerUsage `json:"usage"`
}



type MetadataPod struct {
	Name string `json:"name"`
	Namespace string `json:"namespace"`
	SelfLink string `json:"selfLink"`
	CreationTimestamp string `json:"creationTimestamp"`
}




type Usage struct{
	Cpu string `json:"cpu"`
	Memory string `json:"memory"`
}

type ContainerUsage struct{
	Cpu string `json:"cpu"`
	Memory string `json:"memory"`
	CpuInt int64
	MemoryInt int64
}