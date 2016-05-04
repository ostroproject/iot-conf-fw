function generateToolBar(parent, resource) {
    var actions = function(backUrl, back, go) {
        this.backUrl = backUrl
        
        this.back = function(event) {
            window.location = this.backUrl
        }
        this.go = function(event) {
            window.location = this.url
        }

        back.addEventListener("click", this.back.bind(this), false)
        for (var i in go) {
            go[i].addEventListener("click", this.go, false)
        }
    }
    
    var urlBase = window.location.protocol + "//" + window.location.host + "/confs"

    var bar = document.createElement("div")
    bar.className = "toolbar"

    var back = document.createElement("div")
    back.id = "back-button"
    back.className = "left-arrow"
    bar.appendChild(back)

    var naviDiv = document.createElement("div")
    naviDiv.className = "navigator"
    
    var naviSpan = document.createElement("span")
    naviSpan.className = "navigator"
    
    var rl = resource.split("/")
    var go = []

    var root = generateNavigatorButton(naviSpan, "confs", "/ ", urlBase)
    go.push(root)
    if (rl.length < 3) {
        root.disabled = true
    }
    
    for (var i = 2;  i < rl.length;  i++) {
        var url = urlBase + "/" + rl.slice(2,i+1).join("/")
        var butt = generateNavigatorButton(naviSpan, rl[i], " / ", url)

        if (i == rl.length - 1) {
            butt.disabled = true
        }
        
        go.push(butt)
        
        naviSpan.appendChild(butt)
    }
    
    actions("/confs/" + rl.slice(2,rl.length-1).join("/"), back,  go)
    
    naviDiv.appendChild(naviSpan)
    bar.appendChild(naviDiv)
    
    parent.appendChild(bar)
}

function generateNavigatorButton(parent, name, sep, url) {
    parent.appendChild(document.createTextNode(sep))
    
    var butt = document.createElement("button")
    butt.type = "button"
    butt.className = "navigator"
    butt.url = url
    butt.appendChild(document.createTextNode(name))
    
    parent.appendChild(butt)

    return butt
}
