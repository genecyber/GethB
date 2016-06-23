package rpc

import (
    "bytes"
    "fmt"
    "github.com/robertkrimen/otto"
    "log"
    "net/http"
    "os"
    "encoding/json"
    "github.com/fatih/color"
    "github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

/// --- Struct representing plugins.js
var plugins struct {    
    Plugins []plugin `json:"plugins"`
}

type plugin struct {
    ServerMode bool `json:"enabled"`
    Filename  string `json:"filename"`
    Description  string `json:"description"`
    Target string `json:"target"`
}

type RuntimeItem struct {
        Runtime *otto.Otto
        Plugin plugin
}

type Runtimes struct {
        Items []RuntimeItem
}

var loaded = false
var runtimes = Runtimes{[]RuntimeItem{}}

// Load the plugin file.  If the file does not exist
// then return a nil runtime
func loadPluginRuntime() Runtimes {   
    items := []RuntimeItem{}
    _runtimes := Runtimes{items}
    if loaded {
        //color.Red("Plugins should only load once!")
        return runtimes
    }
    /// --- Load JSON
    pluginFile, err := os.Open("plugins.json")
    if err != nil {
        color.Red("-----> Error opening plugins file", err.Error())
        return runtimes
    }
    /// Parse JSON
    jsonParser := json.NewDecoder(pluginFile)
    if err = jsonParser.Decode(&plugins); err != nil {
        color.Red("-----> Error parsing plugins file", err.Error())
        return runtimes
    }
    /// Loop over plugins

    for _,element := range plugins.Plugins {
        runtime := otto.New()
        color.Set(color.FgGreen)
        glog.V(logger.Info).Infoln(element.Filename)        
        color.Set(color.FgHiGreen)  
        glog.V(logger.Info).Infoln(element.Description) 
        color.Unset()
        f, err := os.Open(element.Filename)
        if err != nil {
            if os.IsNotExist(err) {
                return runtimes
            }
            log.Fatal(err)
        }
        defer f.Close()
        buff := bytes.NewBuffer(nil)

        if _, err := buff.ReadFrom(f); err != nil {
            log.Fatal(err)
        }
        
        // Load the plugin file into the runtime before we
        // return it for use
        if _, err := runtime.Run(buff.String()); err != nil {
            log.Fatal(err)
        }
        color.Red(buff.String())
        item := RuntimeItem{runtime, element}    
        _runtimes.AddItem(item)
    }

    loaded = true
    color.Red("%v",len(_runtimes.Items))
    runtimes = _runtimes
    return _runtimes
}

func (runtimes *Runtimes) AddItem(item RuntimeItem) []RuntimeItem {
        runtimes.Items = append(runtimes.Items, item)
        return runtimes.Items
}

func load(name string) bool {
    //runtimes := loadPluginRuntime()
    if len(runtimes.Items) == 0 {
        return false
    }
    for _,element := range runtimes.Items {        
        if element.Plugin.Target == name {
            // By convention we will require plugins have a set name
            result, err := element.Runtime.Call("init", nil)
            if err != nil {
                log.Fatal(err)
            }
            color.Set(color.FgGreen)  
            glog.V(logger.Info).Infoln("loading ", element.Plugin.Filename, " for service: ", name)
            color.Set(color.FgHiGreen)
            glog.V(logger.Info).Infoln(result, element.Runtime) 
            color.Unset()         
        }
    }    
    return true
}

func checkRequest(r *http.Request) bool {
    runtimes := loadPluginRuntime()

    // If we don't have a runtime all requests are accepted
    if len(runtimes.Items) > 0 {
        return true
    }
    for _,element := range runtimes.Items {
        v, err := element.Runtime.ToValue(*r)
        if err != nil {
            log.Fatal(err)
        }

        // By convention we will require plugins have a set name
        result, err := element.Runtime.Call("checkRequest", nil, v)
        if err != nil {
            log.Fatal(err)
        }
        // If the js function did not return a bool error out
        // because the plugin is invalid
        out, err := result.ToBoolean()
        if err != nil {
            log.Fatalf("\"checkRequest\" must return a boolean. Got %s", err)
            return false
        }
        if !out {
            return false
        }
    }
    return true
}

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if checkRequest(r) {
            fmt.Fprintf(w, "Welcome in\n")
        } else {
            w.WriteHeader(http.StatusUnauthorized)
            fmt.Fprintf(w, "Your not allowed!\n")
        }
    })

    if err := http.ListenAndServe(":8080", nil); err != nil {
        panic(err)
    }
}