# mqtt2influx bridge

... pushes mqtt messages to an influxdb. you can configure multiple topics, tags and the field. 

## build

you can build this using a docker container so you do not need to have go installed:

    make build

crosscompiling (see https://golang.org/doc/install/source#environment):
    
    make build GOOS=linux GOARCH=arm
    
## configure

atm you have to use a config.toml file.

    [mqtt]
        host = "tcp://127.0.0.1:1883"
        user = "YO"
        password = "PO"
        topic = "#" 
    [influx]
        host = "http://localhost:8086"
        user = "muh"
        password = "fnord"
        database = "mydb"
        interval = 5    # in seconds
    [[sync]]
        pattern = "foo\\/bar\\/(?P<SENSOR_ID>\\w+)\\/\\w+"
        measurement = "yolo"
    [[sync]]
        pattern = "muh\\/baz\\/(?P<SENSOR_ID>\\w+)\\/loom\\/\\w+"
        measurement = "yolo"
        
you have to provide a regex as pattern where you sadly have to extra (in addition to the regex escaping) escape the \ char.

## run

default uses ./config.toml

    mqtt2influx
    
you can provide a config file path like

    mqtt2influx -file /path/to/success.toml
    
atm some things can be set via flags (check `mqtt2influx -h`), but not all. i'll work on that.

## examples

you can find several input/output examples inside `mqtt2influx_test.go`.

### 1.
    mqtt message: `foo/bar/34/distance 1.345`
    pattern: `foo\\/bar\\/(?P<SENSOR_ID>\\w+)\\/\\w+`
    measurement: `yolo`
    results in influx measurement `yolo` like that:
    
    > select * from yolo
    name: yolo
    time                SENSOR_ID distance
    ----                --------- --------
    1498763970000000000 34        1.345

in this case SENSOR_ID will be a tag and distance will be a field.

### 2.
    mqtt message: `foo/bar/sales/baz/distance 1.345`
    pattern: `foo\\/bar\\/(?P<DEPARTMENT>\\w+)\\/\(\w+\)\/\\w+`
    measurement: `yolo`
    results in influx measurement `yolo` like that:
    
    > select * from yolo
    name: yolo
    time                DEPARTMENT baz
    ----                ---------  ---
    1498763970000000000 sales      1.345
    
in this case DEPARTMENT will be a tag and baz will be a field.

### 3. non numeric and non boolean payloads

if a non numeric or non boolean payload is present we map the payload to a tag and set the "occurred" field to "true"
 
    mqtt message: `foo/bar muh`
    pattern: `foo\\/\\w+`
    measurement: `yolo`
    results in influx measurement `yolo` like that:
    
    > select * from yolo
    name: yolo
    time                bar occurred
    ----                --- ---
    1498763970000000000 muh true
     

# todo

- better error handling
- more tests
- TLS support