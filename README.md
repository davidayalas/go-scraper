#Scraping an application with concurrent requests

This is a POC of Golang and the use of routines and channels and other features in go. 

Bonus track: it inserts the resulting html scrapped items to MongoDB in MongoLab.

The basics of the process is to start a navigation on the "service" that holds de gas stations in Spain:

- The base url is http://geoportalgasolineras.es/searchAddress.do?nomMunicipio=&rotulo=&tipoVenta=false&nombreVia=&numVia=&codPostal=&economicas=false&tipoBusqueda=0&ordenacion=A&posicion={{pos}}&tipoCarburante={{type}}&nomProvincia={{prov}} and it has some variable parameters:

	* nomProvincia (province) is an integer from 1 to 52
	* tipoCarburante (type of gas/fuel) is an integer in [1,3,4,5,6,7,8,15,17,18]

- The process starts the pagination on each url from 1 to 52 and for each type (52 x 10 types = 520 init pages)

- In each page (html) it parses the content an gets the number of stations in that combination of province and type and starts a routine for each page (each page returns 10 records)

- When the response channel gets the body of the page, it parses to get the stations items and puts them in a map.

- When all pagination is done, the items map is saved to a JSON file and inserted in a remote MongoDB.
