package main

import (
    "fmt"
    "io/ioutil"
    "sync"
    "time"
    "strconv"
    "strings"
    "encoding/json"
    "go-scraper/utils"
)

//gas station item data
type gasStation struct {
    Id          string
    Province    string 
    Locality    string
    Address     string
    Date        string
    Name        string
    Location    geoPoint
    Types       map[string]float64
    FullText    string
}

type geoPoint struct{
    Type string 
    Coordinates [2]float64 
}


//Creates an url with the right parameters for pagination, Province and type of fuel/gas
func createURL(u string, pos string, prov string, tyype string) string {
    u = strings.Replace(u, "{{prov}}", prov, -1)
    u = strings.Replace(u, "{{type}}", tyype, -1)
    u = strings.Replace(u, "{{pos}}", pos, -1)
    return u
}


//Converts the map of stations into a json array and array of objects
func getJson(items map[string]gasStation) (string, []interface{}) {

    var docitems []interface{} = make([]interface{}, len(items))
    v := make([]gasStation, len(items))
    i := 0

    for  _, value := range items {
       v[i] = value
       docitems[i] = v[i]
       i++
    }
    jjson, _ := json.Marshal(v)
    return string(jjson), docitems
}



//Read stations from each page
func readStations(responses chan string, response string, items map[string]gasStation, typees map[string]string, wg *sync.WaitGroup, c1 *int, c2 *int, baseurl string, delay *time.Duration){
    
    var nstations, stations, nprov, ntype, aux, aux2 string
    tokens := strings.Split(response, "<p>")

    //test a valid response string
    if strings.Index(response," estaciones")==-1 || strings.Index(tokens[1],"</p>")==-1 {
        return   
    }

    //get number of stations from string in text to run the pagination loop if we are in the first page of a request for a Province and type
    nstations = tokens[1][0:strings.Index(tokens[1],"</p>")]
    nstations = nstations[0:strings.Index(nstations," estaciones")]
    nstations = nstations[strings.LastIndex(nstations, " ")+1:]
    
    //parse html to get items
    stations = tokens[1][strings.Index(tokens[1],"<tbody>"):]
    stations = stations[0:strings.LastIndex(stations, "</tbody>")]

    tr := strings.Split(stations, "<tr ")

    if len(tr)==0 {
        return
    }

    aux = response[strings.LastIndex(response,"||")+2:]
    nprov = aux[0:strings.Index(aux,"|")]    
    ntype = aux[strings.LastIndex(aux,"|")+1:]    

    ajustCols := 0
    if ntype=="8" || ntype=="16" {
        ajustCols = 1
    }

    //starts routines for pages
    if strings.Index(response, "{{first-request-}}")>-1{
        if n, err := strconv.Atoi(nstations); err==nil {
            n = n/10 // pages have 10 records
            for k:=1; k<n+1; k++ {
                wg.Add(1)
                *c1++    
                *c2++
                go utils.Get(createURL(baseurl, strconv.Itoa(k*10), nprov, ntype), "||"+nprov+"|"+ntype, responses, wg, delay)
                time.Sleep(*delay * time.Millisecond)
            }
        }
    }
    
    getContent := func(s string, s1 string, s2 string) string{
        if strings.Index(s,s1)>-1 && strings.Index(s,s2)>-1{
            return strings.TrimRight(strings.TrimLeft(s[strings.Index(s,s1)+1:strings.Index(s,s2)], " "), " ")
        }
        return ""
    }

    for i:=1; i<len(tr); i++ {
        props := strings.Split(tr[i],"<td ")

        station := gasStation{}
        station.Province = getContent(props[1],">","<")
        station.Address = getContent(props[3],">","<")

        aux = getContent(props[11+ajustCols],">", "</img>")
        if len(aux)>0 {
            aux = aux[strings.LastIndex(aux,"(")+1:strings.LastIndex(aux,")")]
            aux2 = strings.Replace(aux[0:strings.Index(aux,",")],",",".",-1)
            if faux64, err := strconv.ParseFloat(aux2, 64); err == nil {
                station.Location.Coordinates[1] = faux64
            }
            aux2 = strings.Replace(aux[strings.Index(aux,",")+1:strings.LastIndex(aux,",")],",",".",-1)
            if faux64, err := strconv.ParseFloat(aux2, 64); err == nil {
                station.Location.Coordinates[0] = faux64
            }
            station.Location.Type = "Point"
        }

        if _, ok := items[station.Province+station.Address+strconv.FormatFloat(station.Location.Coordinates[0],'f',10,64)]; ok{
            station = items[station.Province+station.Address+strconv.FormatFloat(station.Location.Coordinates[0],'f',10,64)]
        }    

        station.Locality = getContent(props[2],">","<")
        station.Date = getContent(props[5],">","<")
        if len(station.Types)==0 {
            station.Types = make(map[string]float64)
        }
        station.Name = getContent(props[7+ajustCols],">","<")
        station.Id = strconv.FormatFloat(station.Location.Coordinates[0],'f',10,64) + "|" + strconv.FormatFloat(station.Location.Coordinates[1],'f',10,64)
        aux = strings.Replace(getContent(props[6],">","<"),",",".",-1)
        if faux64, err := strconv.ParseFloat(aux, 64); err == nil{ 
            station.Types[typees[ntype]] = faux64
        }
        station.FullText = utils.FlattenWord(" " + station.Name + " " + station.Locality + " " + station.Address + " " + station.Province + " ")
        items[station.Province+station.Address+strconv.FormatFloat(station.Location.Coordinates[0],'f',10,64)] = station
    }   
}


func main() {

    start := time.Now()

    baseurl := "http://geoportalgasolineras.es/searchAddress.do?nomMunicipio=&rotulo=&tipoVenta=false&nombreVia=&numVia=&codPostal=&economicas=false&tipoBusqueda=0&ordenacion=A&posicion={{pos}}&tipoCarburante={{type}}&nomProvincia={{prov}}"

    //maps of Types fuel/gases in Spain
    typees := map[string]string{
        "1":"Gasolina_95",
        "3":"Gasolina_98",
        "4":"Gasoleo_A_habitual",
        "5":"Nuevo_gasoleo_A",
        "6":"Gasoleo_B",
        "7":"Gasoleo_C",
        "8":"Biodiesel",
        "15":"Bioetanol",
        "17":"Gases_licuados_del_petroleo",
        "18":"Gas_natural_comprimido",
    }

    responses := make(chan string)

    items := make(map[string]gasStation)

    var wg sync.WaitGroup

    Provinces := 52 //number of provinces to retrieve

    var delay time.Duration = 10

    count, count2 := 0,0

    //starts the init pages for a province a gas/fuel type
    for i := 0; i < Provinces; i++ {
        for k,_:= range typees{
            wg.Add(1)
            count++
            count2++

            //the {{firts-request-}} string will notice the response reader that it has to start N processes with the pagination
            go utils.Get(createURL(baseurl, "0", strconv.Itoa(i+1), k), "{{first-request-}}||" + strconv.Itoa(i+1) + "|"+k, responses, &wg, &delay) 
            time.Sleep(delay * time.Millisecond)
        }
    }

    //Read in the responses channel and parses the content to get the 10 stations per page
    for response := range responses{
        count--
        readStations(responses, response, items, typees, &wg, &count, &count2, baseurl, &delay)
        fmt.Print(".")
        if count==0 {
            for x:=0; x<count2; x++{
                wg.Done()
            }
            close(responses)
        }
    }

    wg.Wait()

    jsonstring, docitems := getJson(items)
    ioutil.WriteFile("./items.json", []byte(jsonstring), 0644)

    elapsed := time.Since(start)

    fmt.Printf("\nItems: %d \n", len(items))
    fmt.Printf("It took: %s \n", elapsed)
    fmt.Printf("Number of pages: %d \n", count2)

    utils.UpdateMongo("ds031721.mongolab.com", 31721,"services", "gasolineras", "**********", "**********", docitems)
}