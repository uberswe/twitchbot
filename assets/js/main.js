const messages = document.getElementById ('messages');
const commands = document.getElementById('commands')

let ws = new WebSocket ('wss://' + window.location.host + '/ws');
if (location.protocol !== 'https:')
{
    ws = new WebSocket ('ws://' + window.location.host + '/ws');
}
ws.addEventListener ('message', function (e) {
    let msg = JSON.parse (e.data);
    // {
    //    "key":"notice",
    //    "private_message":{"User":{"ID":"23198345","Name":"im_slaughter","DisplayName":"Im_Slaughter","Color":"#1E90FF","Badges":{"subscriber":3}},"Raw":"@badge-info=subscriber/5;badges=subscriber/3;color=#1E90FF;display-name=Im_Slaughter;emotes=;flags=;id=45b903a3-369d-4bcf-b4dd-6719596c4b9a;mod=0;room-id=158130480;subscriber=1;tmi-sent-ts=1577843394570;turbo=0;user-id=23198345;user-type= :im_slaughter!im_slaughter@im_slaughter.tmi.twitch.tv PRIVMSG #tweak :@mario_espo i swear this event is gonna make this game explode in popularity","Type":1,"RawType":"PRIVMSG","Tags":{"badge-info":"subscriber/5","badges":"subscriber/3","color":"#1E90FF","display-name":"Im_Slaughter","emotes":"","flags":"","id":"45b903a3-369d-4bcf-b4dd-6719596c4b9a","mod":"0","room-id":"158130480","subscriber":"1","tmi-sent-ts":"1577843394570","turbo":"0","user-id":"23198345","user-type":""},"Message":"@mario_espo i swear this event is gonna make this game explode in popularity","Channel":"tweak","RoomID":"158130480","ID":"45b903a3-369d-4bcf-b4dd-6719596c4b9a","Time":"2020-01-01T02:49:54.57+01:00","Emotes":null,"Bits":0,"Action":false}}
    if (msg.key === "message" | msg.key === "notice") {
        messageReceive (msg.private_message);
    } else if (msg.key === "channel") {
        channelReceive (msg.channel);
    } else if (msg.key === "endchannel") {
        channelEndReceive (msg.value);
    } else if (msg.key === "addcommand") {
        channelEndReceive (msg.value);
    } else if (msg.key === "removecommand") {
        channelEndReceive (msg.value);
    } else if (msg.key === "state") {
        userStateHandler (msg.state);
    } else {
        console.log ("message not processed: ")
        console.log (msg)
    }
});

ws.addEventListener ('open', function (e) {
    console.log("socket open")
});

ws.addEventListener ('close', function (e) {
    console.log ("socket closed")
});

document.getElementById ("createcommand").addEventListener ('click', function (e) {
    e.preventDefault ();
    let jsonmsg = JSON.stringify ({
        key: "createcommand",
        command: document.getElementById ("command").value,
        text: document.getElementById ("commandtext").value
    })
    console.log(jsonmsg)
    ws.send (jsonmsg);
});

function removeCommandClicked(e) {
    e.preventDefault ();
    let cmd = e.currentTarget.parentNode.querySelector('p').innerText;
    console.log({key: "removecommand", text: cmd});
    ws.send (JSON.stringify ({key: "removecommand", text: cmd}));
}

function appendMessage(m, c) {
    var message = document.createElement ('div');
    message.className = c;
    message.innerHTML = m;
    messages.appendChild (message);
}

function receiveMessage(message, className) {
    // Prior to getting your messages.
    let shouldScroll = messages.scrollTop + messages.clientHeight === messages.scrollHeight;
    /*
     * Get your messages, we'll just simulate it by appending a new one syncronously.
     */
    appendMessage (message, className);
    // After getting your messages.
    if (!shouldScroll) {
        scrollToBottom ();
    }
}

function messageReceive(obj) {
    if (obj.User.Color !== "") {
        receiveMessage ("<b><span style=\"color:" + obj.User.Color + "\">" + obj.User.DisplayName + ":</span></b> " + obj.Message, 'message')
    } else {
        receiveMessage ("<b>" + obj.User.DisplayName + ":</b> " + obj.Message, 'message')
    }
}

function channelReceive(message) {
    receiveMessage ("Connected to channel " + message, 'channel')
}

function channelEndReceive(message) {
    receiveMessage ("Disconnected from channel " + message, 'channel')
}

function scrollToBottom() {
    messages.scrollTop = messages.scrollHeight;
}

function appendCommand(command) {
    var p = document.createElement ('p');
    var b = document.createElement ('b');
    b.innerText = command["input"];
    p.appendChild(b);
    var c = document.createElement ('div');
    c.className = "cmd";
    c.appendChild(p);
    var p2 = document.createElement('p');
    p2.innerText = command["output"];
    c.appendChild(p2);
    var button = document.createElement('button');
    button.type = "submit";
    button.className = "btn btn-danger";
    button.id = "removecommand";
    button.innerText = "Remove";
    button.addEventListener('click',  removeCommandClicked);
    c.appendChild(button);
    commands.appendChild (c);
}

function clearCommands() {
    commands.innerHTML = "";
}

function userStateHandler(state) {
    console.log("User state received");
    console.log(state);
    if (state["commands"] !== undefined) {
        clearCommands();
        state["commands"].forEach(function (value, index, array) {
            appendCommand(value);
        });
    }
}

scrollToBottom ();