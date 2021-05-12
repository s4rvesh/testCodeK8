package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"time"
)

func main(){

	fmt.Println("savvy K8s running")

	var PodResponseObject PodMetrics
	var NodeResponseObject NodeMetrics

	for i:=0;i<20;i++{

		PodResponseObject = GetPods()
		NodeResponseObject = GetNodes()
		PodResponseObject, NodeResponseObject = GetIntVals(PodResponseObject,NodeResponseObject)
		CheckThresholdPod(PodResponseObject)
		CheckThresholdNode(NodeResponseObject)
		MongoStore(PodResponseObject, NodeResponseObject)

		time.Sleep(10 * time.Second)

	}

}



////////////////////////////////////////////////////////


func GetPods() PodMetrics{

	url := "http://127.0.0.1:8080/apis/metrics.k8s.io/v1beta1/namespaces/default/pods"
	responseData := Getdata(url)

	var PodResponseObject PodMetrics
	json.Unmarshal(responseData, &PodResponseObject)


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
		for j:=0;j<len(PodResponseObject.Pods[i].Containers);j++{
			var TotalPodCpu int64
			var TotalPodMem int64
			for k:=0;k<len(PodResponseObject.Pods[i].Containers[j].ContainerUsages);j++{
				PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].CpuInt,PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].MemoryInt = convertInt(PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].Cpu, PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].Memory)

				TotalPodCpu =TotalPodCpu + PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].CpuInt
				TotalPodMem =TotalPodMem + PodResponseObject.Pods[i].Containers[j].ContainerUsages[k].MemoryInt
			}
			PodResponseObject.Pods[j].Cpu=TotalPodCpu
			PodResponseObject.Pods[j].Memory=TotalPodMem

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

	return cpuMetricsInt/1000000,(memoryMetricsInt/1024)

}
////////////////////////////////////////////////////////////

func CheckThresholdNode(NodeResponseObject NodeMetrics){

	for i:=0;i<len(NodeResponseObject.Nodes);i++{
		if NodeResponseObject.Nodes[i].NodeUsages.CpuInt > 1000{
			MailAlert("Node",NodeResponseObject.Nodes[i].MetadataNodes.Name,"cpu",NodeResponseObject.Nodes[i].NodeUsages.CpuInt )

		} else if NodeResponseObject.Nodes[i].NodeUsages.MemoryInt > 1000{
			MailAlert("Node",NodeResponseObject.Nodes[i].MetadataNodes.Name,"memory",NodeResponseObject.Nodes[i].NodeUsages.MemoryInt)

		}
	}

}


func CheckThresholdPod(PodResponseObject PodMetrics){


	for i:=0;i<len(PodResponseObject.Pods);i++{

				if PodResponseObject.Pods[i].Cpu > 1000000000{

					MailAlert("Pod",PodResponseObject.Pods[i].MetadataPods.Name,"cpu",PodResponseObject.Pods[i].Cpu)

				} else if PodResponseObject.Pods[i].Memory> 1000000000 {

					MailAlert("Pod",PodResponseObject.Pods[i].MetadataPods.Name,"cpu",PodResponseObject.Pods[i].Memory)

				}
	}



}

////////////////////////////////////////////////////////////////////



func (s *smtpServer) Address() string {
	return s.host + ":" + s.port
}

func MailAlert(item string,item_name string, metric_type string, metric_val int64){

	from := "cmpe272team18@gmail.com"
	password := "Kubernetes@cmpe"



	// Receiver email address to be set

	to := []string{
		"sarvesh.upadhye@gmail.com",
	}

	smtpServer := smtpServer{host: "smtp.gmail.com", port: "587"}



	var message []byte
	if item=="node"{
		if metric_type=="memory"{
			m1:="The memory usage of Node: "
			m2:=item_name
			m3:=" is above threshold. Memory Usage:"
			m4:=metric_val
			m5:="Mi"
			message=[]byte(m1+m2+m3+strconv.FormatInt(m4, 10)+m5)

			auth := smtp.PlainAuth("", from, password, smtpServer.host)
			err := smtp.SendMail(smtpServer.Address(), auth, from, to, message)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			m1:="The CPU usage of Node:"
			m2:=item_name
			m3:=" is above threshold. CPU Usage:"
			m4:=metric_val
			m5:="mCores"
			message=[]byte(m1+m2+m3+strconv.FormatInt(m4, 10)+m5)

			auth := smtp.PlainAuth("", from, password, smtpServer.host)
			err := smtp.SendMail(smtpServer.Address(), auth, from, to, message)
			if err != nil {
				fmt.Println(err)
			}

		}


	} else {
		if metric_type=="memory"{
			m1:="The memory usage of Pod:"
			m2:=item_name
			m3:=" is above threshold. Memory Usage:"
			m4:=metric_val/1024
			m5:="Mi"
			message=[]byte(m1+m2+m3+strconv.FormatInt(m4, 10)+m5)

			auth := smtp.PlainAuth("", from, password, smtpServer.host)
			err := smtp.SendMail(smtpServer.Address(), auth, from, to, message)
			if err != nil {
				fmt.Println(err)
			}

		} else {
			m1:="The CPU usage of Pod:"
			m2:=item_name
			m3:=" is above threshold. CPU Usage:"
			m4:=metric_val
			m5:="mCores"
			message=[]byte(m1+m2+m3+strconv.FormatInt(m4, 10)+m5)
			message=[]byte(m1+m2+m3+strconv.FormatInt(m4, 10))

			auth := smtp.PlainAuth("", from, password, smtpServer.host)
			err := smtp.SendMail(smtpServer.Address(), auth, from, to, message)
			if err != nil {
				fmt.Println(err)
			}

		}
	}


}
////////////////////////////////////////////////////////////////////////

func MongoConnect(uri string) (*mongo.Client, context.Context){

	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	return client,ctx
}



func MongoInsert(client *mongo.Client,ctx context.Context,PodResponseObject PodMetrics, NodeResponseObject NodeMetrics) bool{


	col := client.Database("kubernetes-metrics").Collection("custAppMetrics2")

	colPod := client.Database("kubernetes-metrics").Collection("PodsCollectionName")


	currentTime := time.Now()


	for i:=0; i<len(NodeResponseObject.Nodes);i++{

		if NodeResponseObject.Nodes[i].MetadataNodes.Name=="master"{

			_, insertErr := col.InsertOne(ctx,bson.D{


				{Key: "metrics", Value:bson.D{{Key: "cpu", Value: NodeResponseObject.Nodes[i].NodeUsages.CpuInt},{Key: "memory", Value: NodeResponseObject.Nodes[i].NodeUsages.MemoryInt}}},
				{Key: "nodeMetrics", Value: true},
				{Key: "nodeId", Value: 1},
				{Key: "createdBy", Value: "System"},
				{Key: "createdDate", Value: currentTime.String()},

			})
			if insertErr != nil {
				fmt.Println("InsertOne ERROR:", insertErr)
				os.Exit(1) // safely exit script on error
			}
		}
		if NodeResponseObject.Nodes[i].MetadataNodes.Name=="node2"{
			_, insertErr := col.InsertOne(ctx,bson.D{


				{Key: "metrics", Value:bson.D{{Key: "cpu", Value: NodeResponseObject.Nodes[i].NodeUsages.CpuInt},{Key: "memory", Value: NodeResponseObject.Nodes[i].NodeUsages.MemoryInt}}},
				{Key: "nodeMetrics", Value: true},
				{Key: "nodeId", Value: 2},
				{Key: "createdBy", Value: "System"},
				{Key: "createdDate", Value: currentTime.String()},

			})
			if insertErr != nil {
				fmt.Println("InsertOne ERROR:", insertErr)
				os.Exit(1) // safely exit script on error
			}
		}
		if NodeResponseObject.Nodes[i].MetadataNodes.Name=="node2"{
			_, insertErr := col.InsertOne(ctx,bson.D{


				{Key: "metrics", Value:bson.D{{Key: "cpu", Value: NodeResponseObject.Nodes[i].NodeUsages.CpuInt},{Key: "memory", Value: NodeResponseObject.Nodes[i].NodeUsages.MemoryInt}}},
				{Key: "nodeMetrics", Value: true},
				{Key: "nodeId", Value: 3},
				{Key: "createdBy", Value: "System"},
				{Key: "createdDate", Value: currentTime.String()},

			})
			if insertErr != nil {
				fmt.Println("InsertOne ERROR:", insertErr)
				os.Exit(1) // safely exit script on error
			}
		}

	}


	for i:=0; i<len(PodResponseObject.Pods);i++{


		if PodResponseObject.Pods[i].MetadataPods.Name == "demo-app1"{

			_, insertErr := colPod.InsertOne(ctx,bson.D{
				{Key: "appId", Value:bson.D{{Key: "cpu", Value: PodResponseObject.Pods[i].Containers[i].ContainerUsages},{Key: "memory", Value: NodeResponseObject.Nodes[0].NodeUsages.MemoryInt}}},
				{Key: "nodeMetrics", Value: true},
				{Key: "nodeId", Value: 1},
				{Key: "createdBy", Value: "System"},
				{Key: "createdDate", Value: currentTime.String()},

			})
			if insertErr != nil {
				fmt.Println("InsertOne ERROR:", insertErr)
				os.Exit(1) // safely exit script on error
			} else {
				fmt.Println("Added Node data to mongo")
			}

		}
		if PodResponseObject.Pods[i].MetadataPods.Name=="demo-app2"{




		}


	}














	return true
}

func MongoStore(PodResponseObject PodMetrics,NodeResponseObject NodeMetrics){
	uri:="mongodb+srv://admin:admin123@cluster0.lnxpp.mongodb.net/kubernetes-metrics?retryWrites=true&w=majority"
	client, ctx := MongoConnect(uri)

	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(databases)
	MongoInsert(client,ctx, PodResponseObject,NodeResponseObject)

	defer client.Disconnect(ctx)

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
	Cpu int64
	Memory int64
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



type ContainerUsage struct{
	Cpu string `json:"cpu"`
	Memory string `json:"memory"`
	CpuInt int64
	MemoryInt int64
}