dialogBox = "";

function dottedIP4Pattern() {
        return "[0-9][0-9]{0,2}\.[0-9][0-9]{0,2}\.[0-9][0-9]{0,2}\.[0-9][0-9]{0,2}"
}

function updateForm(resource, name) {
    var xmlhttp = new XMLHttpRequest();

    xmlhttp.onreadystatechange = function() {
        if (xmlhttp.readyState == 4) {
            if (xmlhttp.status == 200) {
                parseValues(JSON.parse(xmlhttp.responseText), "");
            }
            delete xmlhttp
        }
    }

    xmlhttp.open("GET", getRestURL(resource), true);
    xmlhttp.send();
}

function saveForm(resource, name) {
    var values = JSON.stringify(writeValues())
    var xmlhttp = new XMLHttpRequest();

    xmlhttp.onreadystatechange = function() {
        if (xmlhttp.readyState == 4) {
            var status = "values succesfully sent"
            if (xmlhttp.status != 200) {
                if (xmlhttp.responseText != "") {
                    status = xmlhttp.responseText
                }
                else if (xmlhttp.statusText != "") {
                    status = xmlhttp.statusText
                }
                else {
                    status = "failed to send values"
                }
            }
            dialogBox.popup(status)
            delete xmlhttp
        }
    }

    //console.log("values: " + values)
    
    xmlhttp.open("PUT", getRestURL(resource), true);
    xmlhttp.setRequestHeader("Content-Type", "application/json");
    xmlhttp.setRequestHeader("Accept", "application/json; charset=utf-8");
    xmlhttp.send(values);
}

function getRestURL(resource) {
    var origin = window.location.origin.split(":")
    return origin[0] + ":" + origin[1] + ":4984/confs" + resource;
}

function parseValues(values, prefix) {
    for (var key in values) {
        var value = values[key]
        var name = prefix + key
        var elems = document.getElementsByName(name)
                
        if (typeof value != "object") {
            if (elems.length > 0) {
                var elem = elems[0]
                switch (elem.type) {
                case "checkbox":
                    elem.checked = value;
                    break;
                default:
                    elem.value = value;
                    break;
                }

                var evt  = document.createEvent("HTMLEvents")
                evt.initEvent("change", false, true)
                elem.dispatchEvent(evt)
            }
        }
        else {
            if (elems.length > 0) {
                elems[0].value = "-";
            }
            parseValues(value, name + ".");
        }
    }
}

function writeValues() {
    var values = {}
    var elems = document.getElementsByClassName("input")
    
    for (i = 0;  i < elems.length;   i++) {
        var elem = elems[i]
        var key = elem.name
        var value

        switch (elem.type) {
        case "checkbox":
            value = elem.checked
            break
        case "number":
            value = parseInt(elem.value)
            break
        default:
            value = elem.value
            break
        }
        
        if (key && typeof value != "undefined") {            
            addValue(values, key, value)
        }
    }

    return values
}

function addValue(values, key, value) {
    var kl = key.split(".")

    if (kl.length == 1) {
        values[key] = value
        return
    }

    if (kl.length > 1) {
        var k = kl[0]
        var v = values[k]

        switch (typeof v) {
        case "object":
            break;
        default:
            if (typeof v != "string" || v != "-")
                return;
        case "undefined":
            v = {}
            values[k] = v
            break;
        }
        
        addValue(v, kl.splice(1, kl.length - 1).join("."), value)
        return
    }
}


function setInput(id, disabled, required) {
    var elem = document.getElementById(id)
    elem.disabled = disabled;
    elem.required = (disabled ? false : required);
}

function generateWorkArea(parent) {
    var workArea = document.createElement("div")
    workArea.className = "workarea"

    generateForm(workArea, pageDef)
    
    parent.appendChild(workArea)
    return workArea
}

function generateFormEntryTable(form, def) {
    var table = document.createElement("table")
    table.className = "entry"
    
    generateFields(table, def.name, "", 0, def.fields)

    form.appendChild(table)
}

function generateForm(parent, def) {    
    var form = document.createElement("form")
    form.id = def.name + "Entry"
    form.className = "entry"
    form.action = "#"

    var container = document.createElement("div")
    container.className = "container"
    
    generateFormEntryTable(container, pageDef)

    form.appendChild(container)

    generateFormButtonBar(form, pageDef)
    
    parent.appendChild(form)

    return form
}


function generateFields(table, idPrefix, namePrefix, depth, fields)
{
    for (var name in fields) {
        var def = fields[name]
        var id = idPrefix + name
        var fullName = namePrefix + name
        
        var row = document.createElement("tr")
        table.appendChild(row)

        var label = document.createElement("td")
        label.className = "label" + ((depth > 0) ? String(depth) : "")
        label.appendChild(document.createTextNode(name))
        row.appendChild(label)

        var value = document.createElement("td")
        value.className = "value"
        generateValue(table, value, id, fullName, depth, def)
        row.appendChild(value)
    }
}

function generateValue(table, parent, id, name, depth, def) {
    var value
    
    switch (def.type) {
    case "checkbox":
        value = generateInput(parent, id, name, def)
        break
    case "select":
        value = generateSelect(parent, id, name, def)
        break
    case "number":
        def.pattern = "[0-9]+"
    case "text":
        value = generateInput(parent, id, name, def)
        break
    case "password":
        value = generateInput(parent, id, name, def)
        break
    case "section":
        value = generateSection(table, parent, id, name, depth, def)
        break
    }

    if (value && ("events" in def)) {
        for (event in def.events) {
            handler = def.events[event]

            if (typeof handler == "function") {
                value.addEventListener(event, handler, false)
            }
        }
    }

    return value
}

function generateInput(parent, id, name, def) {
    var input = document.createElement("input")
    input.type = def.type
    input.id = id
    input.className = "input"
    input.name = name
    input.title = def.desc

    if ("defval" in def) {
        switch (def.type) {
        case "checkbox":  input.selected = def.defval;  break
        case "text":      input.value = def.defval;     break
        }
    }
    
    if ("pattern" in def)  { input.pattern = def.pattern }
    if ("min" in def)      { input.min = def.min }
    if ("max" in def)      { input.max = def.max }
    if ("size" in def)     { input.size = def.size }

    parent.appendChild(input)

    return input
}

function generateSelect(parent, id, name, def) {
    var select = document.createElement("select")
    select.id = id
    select.className = "input"
    select.name = name
    select.title = def.desc

    for (optVal in def.options) {
        var optText = def.options[optVal]
        
        var option = document.createElement("option")
        option.value = optVal
        option.appendChild(document.createTextNode(optText))
        if (optVal == def.defval) { option.selected = true }

        select.appendChild(option)
    }

    parent.appendChild(select)

    return select
}

function generateSection(table, value, id, name, depth, def) {
    var secValue = false

    if (("value" in def) && (def.value.type != "section")) {
        secValue = generateValue(table, value, id, name, depth, def.value)
    }

    if ("fields" in def) {
        generateFields(table, id, name + ".", depth+1, def.fields) 
    }

    return secValue
}

function generateFormButtonBar(form, def) {
    var actions = function(resource, name, form, reload, reset) {
        this.resource = resource
        this.name = name
        this.form = form

        this.submit = function(event) {
            event.preventDefault()
            saveForm(this.resource, this.name)
        };
        this.reload = function(event) {
            updateForm(this.resource, this.name)
        };
        this.reset = function(event) {
            console.log("reset " + this.resource)
        };
        
        form.addEventListener("submit", this.submit.bind(this), false)
        reload.addEventListener("click", this.reload.bind(this), false)
        reset.addEventListener("click", this.reset.bind(this), false)
    }
    
    var buttonBar = document.createElement("div")
    buttonBar.className = "buttonbar"

    var buttons = document.createElement("div")
    buttons.className = "buttons"

    var apply = document.createElement("button")
    apply.type = "submit"
    apply.className = "buttonbar"
    apply.appendChild(document.createTextNode("Apply"))
    buttons.appendChild(apply)
    
    var reload = document.createElement("button")
    reload.type = "button"
    reload.className = "buttonbar"
    reload.appendChild(document.createTextNode("Reload"))
    buttons.appendChild(reload)
    
    var reset = document.createElement("button")
    reset.type = "button"
    reset.className = "buttonbar"
    reset.appendChild(document.createTextNode("Reset"))
    buttons.appendChild(reset)
    
    actions(def.resource, def.name, form, reload, reset)

    buttonBar.appendChild(buttons)
    form.appendChild(buttonBar)

    return buttonBar
}

function generateDialog(parent, def) {
    var cover = document.createElement("div")
    cover.className = "dialogcover"
    parent.appendChild(cover)
    
    var dialog = document.createElement("div")
    dialog.id = def.name + "Dialog"
    dialog.className = "dialog"

    var container = document.createElement("div")
    container.className = "container"

    generateMsgbox(cover, dialog, container, def)

    dialog.appendChild(container)
    
    parent.appendChild(dialog)
}

function generateMsgbox(cover, dialog, container, def) {
    var msgbox = document.createElement("div")
    msgbox.className = "msgbox"
    container.appendChild(msgbox)

    var message = document.createElement("span")
    message.id = def.name + "Message"
    message.className = "message"
    msgbox.appendChild(message)

    generateMsgboxButtonBar(cover, dialog, msgbox, message, def)
    
    return msgbox
}

function generateMsgboxButtonBar(cover, dialog, msgbox, message, def) {
    var actions  = function(cover, dialog, message, ok) {
        this.cover = cover
        this.dialog = dialog;
        this.message = message;

        this.popup = function(text) {
            console.log("popup")
            this.message.innerHTML = text
            this.cover.style.display = "block"
            this.dialog.style.display = "block"
        };
        this.popdown = function(event) {
            this.cover.style.display = "none"
            this.dialog.style.display = "none"
        };

        ok.addEventListener("click", this.popdown.bind(this), false);
    };
    
    var buttonBar = document.createElement("div")
    buttonBar.className = "buttonbar"

    var buttons = document.createElement("div")
    buttons.className = "buttons"

    var ok = document.createElement("button")
    ok.type = "button"
    ok.className = "buttonbar"
    ok.appendChild(document.createTextNode("OK"))
    buttons.appendChild(ok)
    
    buttonBar.appendChild(buttons)
    
    msgbox.appendChild(buttonBar)

    dialogBox = new actions(cover, dialog, message, ok)

    return buttonBar
}

window.onload = function() {
    var page = document.createElement("div")
    var update

    page.id = pageDef.name + "Page"
    page.className = "page"

    if (pageDef &&
        ("name" in pageDef) &&
        ("resource" in pageDef) &&
        ("title" in pageDef) &&
        ("fields" in pageDef))
    {
        document.title = pageDef.title

        generateToolBar(page, pageDef.resource)
        generateWorkArea(page)
        generateDialog(page, pageDef)
        
        update = true
    }
    else {
        page.appendChild(document.createTextNode(
            "incomplete pageDef: one or more mandatory field is missing"))

        update = false
    }

    document.body.appendChild(page)

    if (update) {
        updateForm(pageDef.resource, pageDef.name)
    }
}
