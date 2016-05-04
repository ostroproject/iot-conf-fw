function generateDirList(parent, def) {
    var rl = def.resource.split("/")
    var name = rl[rl.length - 1]
    
    var dirlist = document.createElement("div")
    dirlist.id = name + "DirList"
    dirlist.className = "dirlist"

    var table = document.createElement("table")
    table.className = "dirlist"

    generateEntries(table, def.resource, def.entries)
    
    dirlist.appendChild(table)
    parent.appendChild(dirlist)

    return dirlist
}

function generateEntries(table, resource, entries) {
    var go = function(event) {
        window.location =  this.url
    }
    var sortedEntries = JSON.parse(JSON.stringify(entries)).sort(function(a,b) {
        var na = a.name
        var nb = b.name

        if (na > nb) { return  1 }
        if (na < nb) { return -1 }

        return 0
    });

    var rl = resource.split("/")
    var path = rl.slice(2, rl.length).join("/")
    var urlBase = window.location.protocol + "//" + window.location.host + "/confs/" + path
    
    for (var i in sortedEntries) {
        var entry = sortedEntries[i]

        var row = document.createElement("tr")

        var label = document.createElement("td")
        label.className = "label"
        label.appendChild(document.createTextNode(entry.desc))
        row.appendChild(label)

        var launcher = document.createElement("td")
        launcher.className = "launcher"
        
        var arrow = document.createElement("div")
        arrow.id = entry.name + "Arrow"
        arrow.className = "right-arrow"
        arrow.url = urlBase + "/" + entry.name
        arrow.addEventListener("click", go, false)
        launcher.appendChild(arrow)
        
        row.appendChild(launcher)

        table.appendChild(row)
    }
}

function generateWorkArea(parent, def) {
    var workArea = document.createElement("div")
    workArea.className = "workarea"

    generateDirList(workArea, def)

    parent.appendChild(workArea)
    return workArea
}


window.onload = function() {
    var page = document.createElement("div")

    page.id = pageDef.name + "Page"
    page.className = "page"

    if (pageDef &&
        ("resource" in pageDef) &&
        ("title" in pageDef) &&
        ("entries" in pageDef))
    {
        document.title = pageDef.title

        generateToolBar(page, pageDef.resource)
        generateWorkArea(page, pageDef)

        update = true
    }
    else {
        page.appendChild(document.createTextNode(
            "incomplete pageDef: one or more mandatory field is missing"))

        update = false
    }

    document.body.appendChild(page)
}
