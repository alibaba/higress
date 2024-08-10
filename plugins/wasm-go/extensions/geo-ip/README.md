1. The functionality of geo-ip plugin
This geo-ip plugin finds the client's geographic information depending on his/her ip address. It searches the geoip database to find which ip segment contains client's ip. After that, it gets the geographic information of the segment. And then, it passes the geographic information to latter wasm plugins. It is able to supply geographic prompt and higress could send client's request with this prompt to LLVM chat api upstream server.


2. Plugin configuration:
ipProtocol: $ip_protocol

$ip_protocol stands for the ip protocol that client's ip is. Currently, this plugin implemented search client's geographic information depending on client's ip with type ipv4. Later it will have the ipv6 functionality. The parameter type is string. It could be configured as "ipv4" or "ipv6".


3. How to retrive the geographic data from this plugin?
The geo-ip plugin calls proxywasm.SetProperty() to set the client's geographic data to the request. The program of wasm plugins behind the geo-ip plugin calls proxywasm.GetProperty() to retrieve the geographic data.


4. How to generate geoCidr.txt?
This plugin supply a function to translate ip segment into cidr in generateCidr directory. And each cidr in the result is unique.



