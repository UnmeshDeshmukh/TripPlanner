package main

import (
	// Standard library packages
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Request struct {
	Starting_from_location_id string   `json:"Id" bson:"Id"`
	Location_ids              []string `json:Location_ids`
}

var mgoSession *mgo.Session
var TRACK_ID_CONSTANT int
var ACCESS_TOKEN string

type Response struct {
	Id           string      `json:"Id" "bson":"id"`
	Name         string      `json:"name" bson:"name"`
	Address      string      `json:"address" bson:"address"`
	City         string      `json:"city" bson:"city"`
	State        string      `json:"state" bson:"state"`
	Zip          string      `json:"zip" bson:"zip"`
	Coordinates  interface{} `json:"coordinates" bson:"cooridnates"`
	Location_ids []string    `json:Location_ids`
}
type TripPlanner struct {
	// Id bson.ObjectId `json:"_id "bson:_id"`
	Id                        string   `json:"Id" "bson":"id"`
	Status                    string   `json:"status" "bson":"status"`
	Starting_from_location_id string   `json:"starting_from_location_id" "bson":"startinglocation"`
	Best_route_location_ids   []string `json:"best_route_location_ids" "bson":"best_route_location_ids"`

	Total_uber_costs    float64 `json:"total_uber_costs" "bson":"total_uber_costs"`
	Total_uber_duration float64 `json:"total_uber_duration" "bson":"total_uber_duration"`
	Total_distance      float64 `json:"total_distance" "bson":"total_distance"`
	// uber_wait_time_eta : 5

}

type PutTripPlanner struct {
	// Id bson.ObjectId `json:"_id "bson:_id"`
	Id                           string   `json:"Id" "bson":"id"`
	Status                       string   `json:"status" "bson":"status"`
	Starting_from_location_id    string   `json:"starting_from_location_id" "bson":"startinglocation"`
	Best_route_location_ids      []string `json:"best_route_location_ids" "bson":"best_route_location_ids"`
	Uber_wait_time_eta           int      `json:"uber_wait_time_eta" "bson":"uber_wait_time_eta"`
	Current_location             string   `json:"current_location" "bson":"current_location"`
	Next_destination_location_id string   `json:"next_destination_location_id" "bson":"next_destination_location_id"`
	Total_uber_costs             float64  `json:"total_uber_costs" "bson":"total_uber_costs"`
	Total_uber_duration          float64  `json:"total_uber_duration" "bson":"total_uber_duration"`
	Total_distance               float64  `json:"total_distance" "bson":"total_distance"`
	// uber_wait_time_eta : 5

}

type TripTracker struct {
	Tracker int `json:"tracker" "bson":"tracker"`
}

type Uberdata struct {
	End_id        string
	Duration      float64
	Distance      float64
	High_Estimate float64
}

type Message struct {
	Start_latitude  string `json:"start_latitude"`
	Start_longitude string `json:"start_longitude"`
	End_latitude    string `json:"end_latitude"`
	End_longitude   string `json:"end_longitude"`
	Product_id      string `json:"product_id"`
}

func postt(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	var duration, total_uber_duration float64
	var distance, high_estimate, total_distance, total_uber_costs float64
	//var product_id string
	u := Request{}
	var resp []Response
	resp = append(resp, Response{})

	json.NewDecoder(r.Body).Decode(&u)

	resp[0].Id = u.Starting_from_location_id
	resp[0].Location_ids = u.Location_ids

	resp = GetLocation(resp[0].Location_ids, resp[0].Id)
	//	fmt.Println(resp)
	var bestroute []Response
	bestroute = append(bestroute, Response{})
	bestroute = GetBestRoute(resp)
	var bestrouteLocation [] string
	for index, _ := range bestroute {

		bestrouteLocation = append(bestrouteLocation,bestroute[index].Id)
		startLocationlat := bestroute[index].Coordinates.(bson.M)["lat"].(float64)
		startLocationlong := bestroute[index].Coordinates.(bson.M)["lng"].(float64)

		if index != len(resp)-1 {
			endLocationlat := bestroute[index+1].Coordinates.(bson.M)["lat"].(float64)
			endLocationlong := bestroute[index+1].Coordinates.(bson.M)["lng"].(float64)

			duration, distance, high_estimate, _ = GetPriceEstimates(startLocationlat, startLocationlong, endLocationlat, endLocationlong)
			//	fmt.Println("This the product id",product_id)
		}

		if index == len(resp)-1 {

			duration, distance, high_estimate, _ = GetPriceEstimates(startLocationlat, startLocationlong, bestroute[0].Coordinates.(bson.M)["lat"].(float64), bestroute[0].Coordinates.(bson.M)["lng"].(float64))
			//	fmt.Println("This the product id",product_id)
		}
		total_uber_costs = total_uber_costs + high_estimate
		total_uber_duration = total_uber_duration + duration
		total_distance = total_distance + distance

	}

	//	fmt.Println(" Total Duration", total_uber_duration, "Total Distance", total_distance, "Total High_Estimate", total_uber_costs)

	resp[0].Location_ids = u.Location_ids
	//fmt.Println(resp)
	tripPlannerResponse := TripPlanner{}
	rand.Seed(time.Now().UTC().UnixNano())
	TRACK_ID_CONSTANT = rand.Intn(5000)
	tripPlannerResponse.Id = strconv.Itoa(TRACK_ID_CONSTANT)
	//TRACK_ID_CONSTANT = TRACK_ID_CONSTANT + 1
	tripPlannerResponse.Status = "planning"

	tripPlannerResponse.Starting_from_location_id = u.Starting_from_location_id
	//	fmt.Println(u.Starting_from_location_id)
	bestrouteLocation = append(bestrouteLocation[:0],bestrouteLocation[1:]...)
	tripPlannerResponse.Best_route_location_ids = bestrouteLocation
	tripPlannerResponse.Total_uber_costs = total_uber_costs
	tripPlannerResponse.Total_uber_duration = total_uber_duration
	tripPlannerResponse.Total_distance = total_distance

	MongoInsert(tripPlannerResponse)

	mgoSession, _ := mgo.Dial("mongodb://goassignment:goassignment@ds057214.mongolab.com:57214/goassignment")

	uj, _ := json.MarshalIndent(tripPlannerResponse, "", "\t")

	if err := mgoSession.DB("goassignment").C("TripPlanner").Update(bson.M{"id": strconv.Itoa(TRACK_ID_CONSTANT)}, bson.M{"$set": bson.M{"tracker": 0}}); err != nil {
		fmt.Println("Insertion Failed")
		panic(err)
	}

	// Write content-type, statuscode, payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "The JSON response received is as follows %s", uj)

}

func GetBestRoute(routes []Response) []Response { //

	var bestroute []Response
	var uberdata []Uberdata
	bestroute = append(bestroute, Response{})
	//uberdata = append(uberdata,Uberdata{})
	//uberdata[0].Start_id = routes[0].Id
	bestroute[0] = routes[0]
	routes = DeletefromOldRoute(routes, routes[0].Id)

	NearbylocationId := ""
	//fmt.Println(len(routes))
	lenroutes := len(routes)
	for i := 0; i < lenroutes; i++ {
		startLocationlat := bestroute[i].Coordinates.(bson.M)["lat"].(float64)
		startLocationlong := bestroute[i].Coordinates.(bson.M)["lng"].(float64)
		//fmt.Println("The bestroute Initially is",bestroute,"\n\n")
		uberdata = uberdata[:0]
		index_uber := 0
		for index, _ := range routes {

			endLocationlat := routes[index].Coordinates.(bson.M)["lat"].(float64)
			endLocationlong := routes[index].Coordinates.(bson.M)["lng"].(float64)
			uberdata = append(uberdata, Uberdata{})

			uberdata[index_uber].Duration, uberdata[index_uber].Distance, uberdata[index_uber].High_Estimate, _ = GetPriceEstimates(startLocationlat, startLocationlong, endLocationlat, endLocationlong)
			uberdata[index_uber].End_id = routes[index].Id
			//fmt.Println("These are the endids.......", uberdata[index1].End_id)
			index_uber++
			//fmt.Println("Uberdata High_Estimate", uberdata[index1].High_Estimate)

			//Delete(routes)
			//fmt.Println("This is the data",uberdata)

		}
		NearbylocationId = Returnlowest(uberdata)
		//fmt.Println("Returned Nearest Location is ", NearbylocationId)
		bestroute = CreateBestRoute(routes, NearbylocationId, bestroute)
		//fmt.Println("The Created bestroute  is", bestroute, "\n\n")
		routes = DeletefromOldRoute(routes, NearbylocationId)
		//fmt.Println("The New Routes Array after deletion is", routes, "\n\n")

	}
	//fmt.Println(uberdata)
	//fmt.Println("This is the data of bestroute------", bestroute, "\n\n")
	//return routes add later
	return bestroute
}

func CreateBestRoute(routes []Response, NearbylocationId string, bestroute []Response) []Response {
	index1 := len(bestroute)
	bestroute = append(bestroute, Response{})
	for index, _ := range routes {
		if NearbylocationId == routes[index].Id {
			bestroute[index1] = routes[index]
		}
	}
	return bestroute
}

func DeletefromOldRoute(routes []Response, NearbylocationId string) []Response {

	for index, _ := range routes {
		if NearbylocationId == routes[index].Id {
			routes = append(routes[:index], routes[index+1:]...)
			break
		}
	}
	return routes
}

func Returnlowest(uberdata []Uberdata) string {
	min := 9999.00
	minduration := 9999.00
	Id := ""
	//fmt.Println(uberdata[index].High_Estimate)
	for index, _ := range uberdata {
		//fmt.Println("The high estimate is",uberdata[index].High_Estimate)
		//fmt.Println("The current min is",min)
		if uberdata[index].High_Estimate < min {
			min = uberdata[index].High_Estimate
			Id = uberdata[index].End_id
			//break
		}else if uberdata[index].High_Estimate == min{
			for index1,_ := range uberdata{
			if uberdata[index1].Duration < minduration{
				minduration = uberdata[index1].Duration
				Id = uberdata[index1].End_id
			}

		}
		}

	}
	
	return Id
}

func GetLocation(location_ids []string, starting_from_location_id string) []Response {
	var number []int
	var startLocation int
	startLocation, _ = strconv.Atoi(starting_from_location_id)
	number = append(number, startLocation)
	var resp []Response
	/*for index1, _ := range location_ids {
		fmt.Println(location_ids[index1])
	}*/

	//resp = append(resp, MongoConnect(startLocation))
	//resp := Response{}
	// resp = MongoConnect(startLocation)
	for _, element := range location_ids {
		temp, _ := strconv.Atoi(element)
		number = append(number, temp)
	}
	for index, _ := range number {
		resp = append(resp, MongoConnect(number[index]))
		temp_location := strconv.Itoa(number[index])

		resp[index].Id = temp_location

	}

	return resp

}

func GetPriceEstimates(start_latitude float64, start_longitude float64, end_latitude float64, end_longitude float64) (float64, float64, float64, string) {
	var Url *url.URL
	Url, err := url.Parse("https://sandbox-api.uber.com")
	if err != nil {
		panic("Error Panic")
	}
	Url.Path += "/v1/estimates/price"
	parameters := url.Values{}
	start_lat := strconv.FormatFloat(start_latitude, 'f', 6, 64)
	start_long := strconv.FormatFloat(start_longitude, 'f', 6, 64)
	end_lat := strconv.FormatFloat(end_latitude, 'f', 6, 64)
	end_long := strconv.FormatFloat(end_longitude, 'f', 6, 64)
	parameters.Add("server_token", "5tyNL5jvocvFaQLfqGbZIyoB0xwMuQlJKVPr0l80")
	parameters.Add("start_latitude", start_lat)
	parameters.Add("start_longitude", start_long)
	parameters.Add("end_latitude", end_lat)
	parameters.Add("end_longitude", end_long)
	Url.RawQuery = parameters.Encode()

	res, err := http.Get(Url.String())
	//fmt.Println(Url.String())
	if err != nil {
		panic("Error Panic")
	}
	defer res.Body.Close()
	//contents, _ := ioutil.ReadAll(res.Body)
	//fmt.Printf("%s\n", contents)
	var v map[string]interface{}
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&v); err != nil {
		fmt.Println("ERROR: " + err.Error())
	}

	duration := v["prices"].([]interface{})[0].(map[string]interface{})["duration"].(float64)
	distance := v["prices"].([]interface{})[0].(map[string]interface{})["distance"].(float64)
	product_id := v["prices"].([]interface{})[0].(map[string]interface{})["product_id"].(string)
	//fmt.Println(product_id)
	high_estimate := v["prices"].([]interface{})[0].(map[string]interface{})["high_estimate"].(float64)

	//fmt.Println("Duration", duration, "Distance", distance, "High_Estimate", high_estimate)

	return duration, distance, high_estimate, product_id
}

func MongoConnect(location int) Response {
	resp := Response{}
	mgoSession, err := mgo.Dial("mongodb://goassignment:goassignment@ds057214.mongolab.com:57214/goassignment")
	// https://api.mongolab.com/api/1/databases?apiKey=NehIyTKy-1dStg0RKySzPjAWpKd39ful
	//mongodb://goassignment:goassignment@ds057214.mongolab.com:57214/goassignment
	// Check if connection error, is mongo running?
	if err != nil {
		fmt.Println("over Here-------------------------")
		panic(err)

	}
	//fmt.Println("Location int is ", location)

	// Fetch user
	if err := mgoSession.DB("goassignment").C("users").Find(bson.M{"id": location}).One(&resp); err != nil {
		//  w.WriteHeader(404)
		fmt.Println(" Or    ---------------over Here-------------------------")
		panic(err)
	}
	return resp
}



func MongoInsert(tripPlannerResponse TripPlanner) {
	mgoSession, err := mgo.Dial("mongodb://goassignment:goassignment@ds057214.mongolab.com:57214/goassignment")
	// Check if connection error, is mongo running?
	if err != nil {
		panic(err)
	}
	mgoSession.DB("goassignment").C("TripPlanner").Insert(tripPlannerResponse)
}

func gett(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	id := p.ByName("id")

	tripObject := TripPlanner{}
	tripObject = MongoSelect(id)

	// Marshal provided interface into JSON structure

	// Write content-type, statuscode, payload

	uj, _ := json.MarshalIndent(tripObject, "", "\t")
	//fmt.Println(uj)
	// Write content-type, statuscode, payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "The JSON response received is as follows %s", uj)
}

func MongoSelect(id string) TripPlanner {
	resp := TripPlanner{}

	mgoSession, err := mgo.Dial("mongodb://goassignment:goassignment@ds057214.mongolab.com:57214/goassignment")

	if err != nil {
		fmt.Println("over Here-------------------------")
		panic(err)

	}
	// Stub user

	// Fetch user
	if err := mgoSession.DB("goassignment").C("TripPlanner").Find(bson.M{"id": id}).One(&resp); err != nil {
		//  w.WriteHeader(404)
		fmt.Println(" Or    ---------------over Here-------------------------")
		panic(err)
	}
	return resp

}

func putt(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	var nextLocation, currentLocation string
	var eta float64
	id := p.ByName("id")
	var flag int
	flag = 0
	//fmt.Println(trip_id)

	mgoSession, err := mgo.Dial("mongodb://goassignment:goassignment@ds057214.mongolab.com:57214/goassignment")
	if err != nil {
		panic(err)
	}

	//Retreiving the tracker

	tripTracker := TripTracker{}

	if err := mgoSession.DB("goassignment").C("TripPlanner").Find(bson.M{"id": id}).One(&tripTracker); err != nil {
		fmt.Println("404 sent from here")
		w.WriteHeader(404)
		return
	}

	//fmt.Println("Tracking pointer is ", tripTracker.Tracker)

	putTripPlanner := PutTripPlanner{}
	if err := mgoSession.DB("goassignment").C("TripPlanner").Find(bson.M{"id": id}).One(&putTripPlanner); err != nil {
		fmt.Println("404 sent")
		w.WriteHeader(404)
		return
	}

	if tripTracker.Tracker == 0 {
		currentLocation = putTripPlanner.Starting_from_location_id
		nextLocation = putTripPlanner.Best_route_location_ids[0]
		if err := mgoSession.DB("goassignment").C("TripPlanner").Update(bson.M{"id": id}, bson.M{"$set": bson.M{"current_location": currentLocation, "next_destination_location_id": nextLocation}}); err != nil {
			fmt.Println("404 sent")
			w.WriteHeader(404)
		}
			if err := mgoSession.DB("goassignment").C("TripPlanner").Update(bson.M{"id": id}, bson.M{"$set": bson.M{"status": "requesting"}}); err != nil {
			fmt.Println("404 sent")
			w.WriteHeader(404)
			return
		}
		tripTracker.Tracker += 1

	} else {
		if tripTracker.Tracker == len(putTripPlanner.Best_route_location_ids) {
			currentLocation = putTripPlanner.Next_destination_location_id
			nextLocation = putTripPlanner.Starting_from_location_id


		} else if tripTracker.Tracker > len(putTripPlanner.Best_route_location_ids) {

			if err := mgoSession.DB("goassignment").C("TripPlanner").Update(bson.M{"id": id}, bson.M{"$set": bson.M{"status": "completed"}}); err != nil {
			fmt.Println("404 sent")
			w.WriteHeader(404)
			return
		}	
			flag =1
			fmt.Println("Trip completed");
			//w.WriteHeader(404)
			//return
			//tripTracker.Tracker += 1		
		} else {
			currentLocation = putTripPlanner.Next_destination_location_id
			nextLocation = putTripPlanner.Best_route_location_ids[tripTracker.Tracker]
		}
		if flag !=1{
		if err := mgoSession.DB("goassignment").C("TripPlanner").Update(bson.M{"id": id}, bson.M{"$set": bson.M{"current_location": currentLocation, "next_destination_location_id": nextLocation}}); err != nil {
			fmt.Println("404 sent")
			w.WriteHeader(404)
			return
		}
	}
			tripTracker.Tracker += 1
 
	}

	if err := mgoSession.DB("goassignment").C("TripPlanner").Update(bson.M{"id": id}, bson.M{"$set": bson.M{"tracker": tripTracker.Tracker}}); err != nil {
		fmt.Println("Insertion Failed")
		panic(err)
	}

	//ETA Retreiving
	var length int
	length = len(putTripPlanner.Best_route_location_ids) + 1
	//fmt.Println("The Trip Tracker is",tripTracker.Tracker)
	//fmt.Println("The length is",length)
 if (tripTracker.Tracker) <= (length){
	eta = GetETA(nextLocation, currentLocation)
}else {
// 	fmt.Println(nextLocation)
	
	eta = 0	
}

	//fmt.Println("ETA found out to be", eta)

	if err := mgoSession.DB("goassignment").C("TripPlanner").Update(bson.M{"id": id}, bson.M{"$set": bson.M{"uber_wait_time_eta": eta}}); err != nil {
		fmt.Println("Insertion Failed")
		panic(err)
	}

	//Preparing the PUT RESPONSE
	if err := mgoSession.DB("goassignment").C("TripPlanner").Find(bson.M{"id": id}).One(&putTripPlanner); err != nil {
		fmt.Println("404 sent")
		w.WriteHeader(404)
		return
	}

	uj, _ := json.MarshalIndent(putTripPlanner, "", "\t")
	//fmt.Println(uj)
	// Write content-type, statuscode, payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "The JSON response received is as follows %s", uj)

}

func GetETA(nextLocation string, currentLocation string) float64 {
	//ETA Retreiving
	var responseArray []Response
	responseArray = append(responseArray, Response{})
	responseArray = GetLocation([]string{nextLocation}, currentLocation)
	startLocationlat := responseArray[0].Coordinates.(bson.M)["lat"].(float64)
	startLocationlong := responseArray[0].Coordinates.(bson.M)["lng"].(float64)
	endLocationlat := responseArray[1].Coordinates.(bson.M)["lat"].(float64)
	endLocationlong := responseArray[1].Coordinates.(bson.M)["lng"].(float64)
	//	GetPriceEstimates(start_latitude, start_longitude, end_latitude, end_longitude)
	//strconv.FormatFloat(startLocationlat, 'f', 6, 64)

	_, _, _, product_id := GetPriceEstimates(startLocationlat, startLocationlong, endLocationlat, endLocationlong)

	v1 := Message{
		Start_latitude:  strconv.FormatFloat(startLocationlat, 'f', 6, 64),
		Start_longitude: strconv.FormatFloat(startLocationlong, 'f', 6, 64),
		End_latitude:    strconv.FormatFloat(endLocationlat, 'f', 6, 64),
		End_longitude:   strconv.FormatFloat(endLocationlong, 'f', 6, 64),
		Product_id:      product_id,
	}

	//fmt.Println("This is the product id for the trip", product_id)
	//	bytearray := "start_latitude:" + strconv.FormatFloat(startLocationlat, 'f', 6, 64) + ",start_longitude:" + strconv.FormatFloat(startLocationlong, 'f', 6, 64) + ",end_latitude:" + strconv.FormatFloat(endLocationlat, 'f', 6, 64) + ",end_longitude:" + strconv.FormatFloat(endLocationlong, 'f', 6, 64) + ",product_id:" + product_id

	//var jsonStr = []byte(`{` + bytearray + `}`)
	//fmt.Println(bytearray)
	jsonStr, _ := json.Marshal(v1)
	//fmt.Println(responseArray)
	//fmt.Println(jsonStr)
	//fmt.Println("New -", bytes.NewBuffer(jsonStr))
	client := &http.Client{}
	r, err := http.NewRequest("POST", "https://sandbox-api.uber.com/v1/requests", bytes.NewBuffer(jsonStr)) // <-- URL-
	//encoded payload
	//	r.PostForm = form

	if err != nil {
		panic(err)
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicmVxdWVzdCJdLCJzdWIiOiJlOTk1ZGQwMi0wMDMyLTQ5YjYtODMyYi1iYzFhZTg3MTA3N2UiLCJpc3MiOiJ1YmVyLXVzMSIsImp0aSI6IjcxMmI2M2U0LWQ4MzUtNDVkZi1hNzA0LTBiYTczYjhlYzFlOSIsImV4cCI6MTQ1MDU3Nzk4MSwiaWF0IjoxNDQ3OTg1OTgxLCJ1YWN0IjoiVWJMZko0dXRkOTZERmhwQmx6bmx1ZlljdTZmZzI0IiwibmJmIjoxNDQ3OTg1ODkxLCJhdWQiOiJmQjh1aGx4S3V2WHVZdlZoeGhqVEg3dzJLdnp6eTk5WCJ9.E8qKSfjC6Ossc-J8gpqhcViW_DlBurPb9J9Cp4bvQsfcyfH0rSWMKF31qJxvGvJ8cVO6j4ImEZWmjJD-4G5IzTERotQsZ_WDgFZ-uXC4uRYR1h8rx82WXejU2j_NsuCKC4iLGoW6yytqe6M8tOugVQZjm8ZCL-ufkLgaOaJq8iHVUY-2Gj_qtl-Qhu7agtmwFKCwzkRcpXq7mcIUQLxQqGA-Gl39iZzfBnv6hLOGIid32MgKDxvHF652tSWcEyImPIwKVf_UhxPtKJM_V9DV5-kSbCQ8rWc3uQZLCNNBikPzv04ypF-27YLazrcmjZkI4C3JF4tXrfzHYRbu7hf9wA")

	//	r.Header.Add("Accept", "application/json")

	//r.Header.Set("Content-Type", "application/json")
	//r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	//r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	resp, _ := client.Do(r)
	defer resp.Body.Close()
	//fmt.Println(r.Header)
	var v map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&v); err != nil {
		fmt.Println("ERROR: " + err.Error())
	}

	//fmt.Println("Request Body", body_req)

	//fmt.Println("response Status:", resp.Status)
	//fmt.Println("response Headers:", resp.Header)
	//fmt.Println(resp)
	//body, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println("response Body:", v["eta"])

	//	GetPriceEstimates(start_latitude, start_longitude, end_latitude, end_longitude)
	return v["eta"].(float64)
}

//curl -H 'Accept: application/json' -X PUT '{"Id":"3279","Location_ids": ["3528","1456"]}' http://localhost:8080/trips/1001/request
//curl -H 'Content-Type: application/json' -X PUT http://localhost:8080/trips/1001/request

func main() {
	TRACK_ID_CONSTANT = 1000
	ACCESS_TOKEN = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicHJvZmlsZSJdLCJzdWIiOiJlOTk1ZGQwMi0wMDMyLTQ5YjYtODMyYi1iYzFhZTg3MTA3N2UiLCJpc3MiOiJ1YmVyLXVzMSIsImp0aSI6ImVhYTk4NWNkLWRjZDEtNDFiOC04OGYwLWVkNjljNTY1YTBlNyIsImV4cCI6MTQ1MDU3NDE2NCwiaWF0IjoxNDQ3OTgyMTYzLCJ1YWN0IjoiWllhMWhZNEFMSm92eWZtNnhwbGJ3TXpuc1VjcHhwIiwibmJmIjoxNDQ3OTgyMDczLCJhdWQiOiJmQjh1aGx4S3V2WHVZdlZoeGhqVEg3dzJLdnp6eTk5WCJ9.PXTcjwiFdWOi7-N6f6Xe6jv3pf5WtSIUvu0QlYce4ITGq8e7Gora1v98WN8lgTFtxnWT48YEJ458l4f6Xg0LrsvZ33BG4Xn2knsllo-2s5XF2LiAwJovikwYufJuAi1ESPCJOHIsaQhC6qevSasaBC-DZjKLJf4l9eqBVQKnfP-0PT7znBAtGnrZk_YE-v_3g6H2HfLNFPKCseGXR9B8UHhkPyFAc0bJCAjJ0Re2OyAy1CE16CmD_XnmXQ_v_ZaNu-LOvol7oHReSiTBBTQnqtNXRFj5qrgOvgpVvsjSxFd0FunvjoaaATUhK2LYGUWVSITA2ZDdrCjvzlvZSXkOSQ"
	r := httprouter.New()

	r.GET("/trips/:id", gett)
	r.POST("/trips", postt)
	r.PUT("/trips/:id/request", putt)
	//r.GET("/locations/:id", gett)
	//r.DELETE("/locations/:id", delete)
	//	r.PUT("/locations/:id", putt)
	server := http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: r,
	}
	server.ListenAndServe()
}

