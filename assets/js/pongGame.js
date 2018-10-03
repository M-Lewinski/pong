var canvas = document.getElementById("game");
canvas.width = 640;
canvas.height = 640;
var ctx = canvas.getContext('2d');
var GameWidth = canvas.width;
var GameHeight = canvas.height;
var angle = 0;

window.requestAnimFrame = (function () {
    return window.requestAnimationFrame ||
			window.webkitRequestAnimationFrame ||
			window.mozRequestAnimationFrame ||
			window.oRequestAnimationFrame ||
			window.msRequestAnimationFrame ||
			function (callback) {
			    window.setTimeout(callback, 1000 / 60);
			};
})();


// var uniqueId = 0;

const MsgSession = "Session";

const MsgMessage = "Message";
const MsgCreatePlayer = "Player";
const MsgCreateRoom = "CreateRoom";
const MsgError = "ErrorMsg";
const MsgRoomInfo     = "RoomInfo";
const MsgAllRooms     = "AllRooms";
const MsgJoinRoom     = "JoinRoom";
const MsgLeaveRoom     = "LeaveRoom";
const MsgReadyPlayer  = "ReadyPlayer";
const ParamPlayerId  = "playerId";
const MsgDeleteRoom = "DeleteRoom";


var playerNameInput = document.getElementById("playerNameInput");


var loginButton = document.getElementById("loginButton");
var socket = new WebSocket("ws://localhost:8080/ws");
var currentRoom = null;
var mainPlayer = null;
var PlayingGame = false;
var gamePlayers = null;

var PlayerPosition = 0;
var ColorTable = [];
ColorTable.push("red","blue","yellow", "green");

var RotationTable = [];
RotationTable.push(0.0,180.0,270.0,90.0);

var AnimationId = null;

var movement = {
    right: false,
    left: false
};

socket.binaryType = 'arraybuffer';

socket.onopen = function () {
    console.log("Created websocket connection");
    var session = {};
    session.Message = MsgSession;
    session.Data = {};
    session.Data.Cookie = getSessionCookie();
    socket.send(JSON.stringify(session));
};

socket.onclose = function (){
    if (AnimationId !== null){
        PlayingGame = false;
        close(AnimationId);
    }
};

socket.onmessage = function (msg){
    if (isByteArray(msg.data)){
        var gI = GameInfoParse(msg.data);
        if (PlayingGame === false){
            // Game.init(gI);
            PlayingGame = true;
            if (currentRoom != null){
                gamePlayers = currentRoom.Players;
            }
            else {
                gamePlayers = null;
            }
            PlayerPosition = CalculatePlayerPosition();
            SendInput();
        }
        GameInfoTable(gI);
        if (gI.AlivePlayers === 0){
            ctx.clearRect(0,0,canvas.width,canvas.height);
            PlayingGame = false;
        }
        else{
            gI = RotateGameInfo(gI);
            Game.draw(gI);
        }
    }
    else{
        var json = JSON.parse(msg.data);
        console.log(JSON.stringify(json));
        if(json.hasOwnProperty(MsgMessage)){
            switch (json.Message) {
                case MsgCreatePlayer:
                    receivedPlayerInfo(json);
                    break;
                case MsgError:
                    receivedError(json);
                    break;
                case MsgRoomInfo:
                    receivedRoomInfo(json.Data, document.getElementById( "allRoomTable"));
                    break;
                case MsgAllRooms:
                    receivedAllRooms(json.Data,document.getElementById( "allRoomTable"));
                    break;
                case MsgDeleteRoom:
                    receivedDeleteRoom(json.Data);
                    break;
            }
        }
    }
};

function isByteArray(array) {
    if (array && array.byteLength !== undefined) return true;
    return false;
}

function readyPlayer() {
    if (currentRoom === null) {
        return
    }
    let msg = createMsg(MsgReadyPlayer);
    socket.send(JSON.stringify(msg));
}

function leaveRoom() {
    if (currentRoom === null){
        return;
    }
    let msg = createMsg(MsgLeaveRoom);
    socket.send(JSON.stringify(msg));
}


function receivedAllRooms(Data,table){
    for (let key in Data){
        console.log(JSON.stringify(Data[key]));
        receivedRoomInfo(Data[key],table)
    }
}

function receivedRoomInfo(room, table){
    if (mainPlayer != null){
        updateCurrentRoom(room);
    }
    addNewRoom(room, table);
}

function joinRoom(roomId){
    let msg = createMsg(MsgJoinRoom);
    msg.Data.Id = roomId;
    socket.send(JSON.stringify(msg));
}

function addNewRoom(room, table) {
    let newRow = document.getElementById(room.Id);
    if (newRow == null){
        newRow = table.insertRow(table.rows.length);
    }
    newRow.setAttribute("id", room.Id);
    let cols = "";
    cols += "<td>" + room.Name + "</td>";
    cols += "<td>" + room.NumberOfPlayers + "</td>";
    cols += "<td>" + room.ReadyCount + "</td>";
    cols += "<td>" + room.MaxPlayers + "</td>";
    cols += "<td>" + room.Life + "</td>";
    let status = '<span  class="positive">Waiting</span>';
    if (room.Playing === true){
        status = '<span class="negative">Playing</span>';
    }
    cols += '<td>' + status + '</td>';
    // cols += '<td><button type="button" class="btn btn-success" onclick="joinRoom()" >Join</button></td>';
    cols += `<td><button type="button" class="btn btn-success" onclick="joinRoom('${room.Id}')">Join</button></td>`;
    newRow.innerHTML = cols;
}

function updateCurrentRoom(room) {
    var nameLabel = document.getElementById("roomNameLabel");
    var lifeLabel = document.getElementById("roomLifeLabel");
    var waitingLabel = document.getElementById("roomWaitingLabel");
    var foundPlayer = false;
    for (let i = 0; i < room.Players.length; i++){
        if (room.Players[i] === null) continue;
        if (room.Players[i].Id === mainPlayer.Id){
            let oldTable = document.getElementById("roomTable");
            let newTable = document.createElement('tbody');
            newTable.setAttribute("id", "roomTable");
            for (let j = 0; j < room.Players.length; j++) {
                let cols = '';
                let cPlay = room.Players[j];
                let newRow = newTable.insertRow(newTable.rows.length);
                if (cPlay != null) {
                    // newRow.setAttribute("id", cPlay.Id);
                    cols += '<td>' + j + '</td>';
                    cols += '<td>' + cPlay.Name + '</td>';
                    let ready = '<td class="negative">no</td>';
                    if (room.Ready[j] === true) {
                        ready = '<td class="positive">yes</td>';
                    }
                    cols += ready;
                }
                else {
                    cols += '<td>' + j + '</td>';
                    cols += '<td>Empty space</td>';
                }
                newRow.innerHTML = cols;
            }
            oldTable.parentNode.replaceChild(newTable,oldTable);
            nameLabel.innerHTML = room.Name;
            lifeLabel.innerHTML = room.Life;
            // waitingLabel.innerHTML = parseInt(room.ReadyCount,10) + "/" + parseInt(room.MaxPlayers,10);
            waitingLabel.innerHTML = room.ReadyCount + "/" + room.MaxPlayers;

            currentRoom = room;
            foundPlayer = true;
            break;
        }
    }
    if (currentRoom != null && foundPlayer === false){
        if (currentRoom.Id === room.Id){
            nameLabel.innerHTML = "NONE";
            lifeLabel.innerHTML = "NONE";
            waitingLabel.innerHTML = "NONE";
            currentRoom = null;
            let oldTable = document.getElementById("roomTable");
            oldTable.innerHTML = "";
        }
    }
}

function GameInfoTable(gameInfo) {
    var oldLifeTable = document.getElementById("lifeTable");
    var newLifeTable = document.createElement('tbody');
    newLifeTable.setAttribute("id", "lifeTable");
    for (let j = 0; j < gameInfo.PlayerCount ;j++){
        let color = ColorTable[j];
        let newRow = newLifeTable.insertRow(newLifeTable.rows.length);
        let cols = '';
        cols += '<td class="'+color+'">' +  gameInfo.Lifes[j] + '</td>';
        newRow.innerHTML = cols;
    }
    oldLifeTable.parentNode.replaceChild(newLifeTable,oldLifeTable);

    var oldPlayerTable = document.getElementById("playerTabel");
    var newPlayerTable = document.createElement('tbody');
    newPlayerTable.setAttribute("id", "playerTabel");
    for (let i=0; i < gameInfo.PlayerCount ;i++){
        if (gameInfo.FinishedPlayers[i] === 0){
            continue;
        }
        let playerIndex = gameInfo.FinishedPlayers[i]-1;
        let color = ColorTable[playerIndex];
        let newRow = newPlayerTable.insertRow(newPlayerTable.rows.length);
        let cols = "";
        cols += '<td>'+ (i+1) +'</td>';
        cols += '<td class="' + color+ '">'+ gamePlayers[playerIndex].Name +'</td>';
        newRow.innerHTML = cols;
    }
    oldPlayerTable.parentNode.replaceChild(newPlayerTable,oldPlayerTable);
}

function receivedDeleteRoom(Data){
    console.log("DeleteingRoom");
    var roomId = Data.Id;
    var nameLabel = document.getElementById("roomNameLabel");
    var lifeLabel = document.getElementById("roomLifeLabel");
    var waitingLabel = document.getElementById("roomWaitingLabel");
    if (currentRoom != null){
        if (currentRoom.Id === roomId){
            nameLabel.innerHTML = "NONE";
            lifeLabel.innerHTML = "NONE";
            waitingLabel.innerHTML = "NONE";
            currentRoom = null;
            let oldTable = document.getElementById("roomTable");
            oldTable.innerHTML = "";
        }
    }
    var tableRow = document.getElementById(roomId);
    if (tableRow !== null){
        tableRow.parentNode.removeChild(tableRow);
    }

}

function receivedError(json){
    window.alert(json.Data.ErrorMsg);
}


function clearCache() {
    document.cookie = "Player" + '=; expires=Thu, 01 Jan 1970 00:00:01 GMT;';
}


function receivedPlayerInfo(json){
    console.log("Received mainPlayer info");
    mainPlayer = json.Data;
    playerNameInput.value = json.Data.Name;
    playerNameInput.disabled = true;
    loginButton.disabled = true;

    var expires = "";
    var date = new Date();
    date.setDate(date.getDate() + 1);
    expires = "expires=" + date.toUTCString();
    document.cookie = "Player"+"=" + json.Data.Id + "; " + expires + "; path=/";
}

function getSessionCookie() {
    var name = "Player"+ "=";
    var decodedCookie = decodeURIComponent(document.cookie);
    var ca = decodedCookie.split(';');
    for(var i = 0; i <ca.length; i++) {
        var c = ca[i];
        while (c.charAt(0) === ' ') {
            c = c.substring(1);
        }
        if (c.indexOf(name) === 0) {
            return c.substring(name.length, c.length);
        }
    }
    return "";
}

function getPlayerIdFromUrl(){
    var url = new URL(window.location);
    var pId = url.searchParams.get(ParamPlayerId);
    return pId;
}

function setPlayerIdToUrl(pId){
    var url = new URL(window.location);
    url.searchParams.set(ParamPlayerId,pId);
    window.location = url.toString();
}

function checkIfPlayerIdUrl(){
    var url = new URL(window.location);
    return url.searchParams.has(ParamPlayerId);
}

function createMsg(message){
    var msg = {};
    msg.Message = message;
    msg.Data = {};
    return msg;
}

function login(){
    console.log("Login in");
    var msg = createMsg(MsgCreatePlayer);
    msg.Data.Name = playerNameInput.value;
    socket.send(JSON.stringify(msg));
}

function createRoom() {
    console.log("Creating new room");
    var msg = createMsg(MsgCreateRoom);
    msg.Data.Name = document.getElementById("roomNameInput").value;
    msg.Data.PlayerCount = parseInt(document.getElementById("playerCountInput").value,10);
    msg.Data.Life = parseInt(document.getElementById("roomLifeInput").value,10);
    socket.send(JSON.stringify(msg));
}

function SendInput(){
    if (PlayingGame === true){
        // let buffer = new ArrayBuffer();
        let array = new Int8Array(1);
        if (movement.right === true){
            array[0]=1;
        }
        else if (movement.left === true){
            array[0]=2;
        }
        else{
            array[0]=0;
        }
        // console.log(array[0]);
        socket.send(array.buffer);
    }
    AnimationId = requestAnimFrame(SendInput);
}

function GameInfoParse(data) {
    var GameInfo = {};
    var index = 0;
    var dataArray = new Uint8Array(data);
    // console.log(dataArray);
    GameInfo.DataSize = dataArray[index++];
    GameInfo.PlayerCount = dataArray[index++];
    GameInfo.AlivePlayers = dataArray[index++];
    GameInfo.Lifes = [];
    let i;
    for (i = 0 ; i < GameInfo.PlayerCount;i++){
        GameInfo.Lifes.push(dataArray[index++]);
    }

    GameInfo.FinishedPlayers = [];
    for (i = 0; i < GameInfo.PlayerCount; i++){
        GameInfo.FinishedPlayers.push(dataArray[index++]);
    }

    GameInfo.BallRadius = GetFloat(index,dataArray,GameInfo.DataSize);
    index += GameInfo.DataSize;

    GameInfo.PlatformWidth= GetFloat(index,dataArray,GameInfo.DataSize);
    index += GameInfo.DataSize;

    GameInfo.PlatformHeight = GetFloat(index,dataArray,GameInfo.DataSize);
    index += GameInfo.DataSize;

    GameInfo.DangerZoneSize= GetFloat(index,dataArray,GameInfo.DataSize);
    index += GameInfo.DataSize;

    GameInfo.SpawnerArrow = {};
    GameInfo.SpawnerArrow.x = GetFloat(index,dataArray,GameInfo.DataSize);
    index += GameInfo.DataSize;
    GameInfo.SpawnerArrow.y = GetFloat(index,dataArray,GameInfo.DataSize);
    index += GameInfo.DataSize;

    GameInfo.Platforms = [];
    for (var j = 0; j < GameInfo.PlayerCount; j++){
        var platPos= {};
        platPos.x = GetFloat(index,dataArray,GameInfo.DataSize);
        index += GameInfo.DataSize;
        platPos.y = GetFloat(index,dataArray,GameInfo.DataSize);
        index += GameInfo.DataSize;

        platPos.w = GameInfo.PlatformWidth;
        platPos.h = GameInfo.PlatformHeight;
        if (j > 1){
            platPos.w = GameInfo.PlatformHeight;
            platPos.h = GameInfo.PlatformWidth;
        }

        GameInfo.Platforms.push(platPos);
    }

    GameInfo.Balls = [];
    for (;index < dataArray.length;){
        var ballPos = {};
        ballPos.x = GetFloat(index,dataArray,GameInfo.DataSize);
        index += GameInfo.DataSize;
        ballPos.y = GetFloat(index,dataArray,GameInfo.DataSize);
        index += GameInfo.DataSize;
        GameInfo.Balls.push(ballPos);
    }
    return GameInfo
}

function CalculatePlayerPosition(){
    if (mainPlayer !== null && currentRoom !== null){
        for (let j = 0; j < currentRoom.MaxPlayers; j++){
            if (currentRoom.Players[j].Id === mainPlayer.Id){
                return j;
            }
        }
    }
}

function RotateGameInfo(gameInfo){
    if (PlayerPosition === 0){
        return gameInfo;
    }
    var rotation = RotationTable[PlayerPosition];
    var radRotation  = rotation*Math.PI/180.0;
    gameInfo.SpawnerArrow = RotateVectorCenter(gameInfo.SpawnerArrow,radRotation);
    for (let j = 0; j < gameInfo.Platforms.length; j ++){
        // console.log(gameInfo.Platforms[j]);
        var w = gameInfo.Platforms[j].w;
        var h = gameInfo.Platforms[j].h;
        gameInfo.Platforms[j] = RotateVectorCenter(gameInfo.Platforms[j],radRotation);
        gameInfo.Platforms[j].w = w;
        gameInfo.Platforms[j].h = h;
        if (PlayerPosition > 1){
            gameInfo.Platforms[j].w = h;
            gameInfo.Platforms[j].h = w;
        }
    }
    for (let j = 0; j < gameInfo.Balls.length; j ++){
        gameInfo.Balls[j] = RotateVectorCenter(gameInfo.Balls[j],radRotation);
    }
    return gameInfo;
}

function RotateVector(vec,rad) {
    var newVec = {};
    newVec.x = Math.cos(rad)*vec.x - Math.sin(rad)*vec.y;
    newVec.y = Math.sin(rad)*vec.x + Math.cos(rad)*vec.y;
    return newVec;
}

function RotateVectorCenter(vec, rad){
    var center = {};
    center.x = canvas.width/2.0;
    center.y = canvas.height/2.0;
    var subVec = {};
    subVec.x = vec.x - center.x;
    subVec.y = vec.y - center.y;
    var rotVec = RotateVector(subVec,rad);
    rotVec.x += center.x;
    rotVec.y += center.y;
    return rotVec;
}

function GetFloat(index,data, datasize){
    var temp = new Uint8Array(datasize);
    for (var j = 0; j < datasize; j++){
        temp[j] = data[index++]
    }
    var result = new Float64Array(temp.buffer);
    return result[0];
}

var Game = {

    draw: function (gameInfo) {
        var dangerAreaDrawTable = [];
        switch (PlayerPosition) {
            case 0:
                dangerAreaDrawTable.push(0,1,2,3);
                break;
            case 1:
                dangerAreaDrawTable.push(1,0,3,2);
                break;
            case 2:
                dangerAreaDrawTable.push(2,3,1,0);
                break;
            case 3:
                dangerAreaDrawTable.push(3,2,0,1);
                break;
        }

        ctx.clearRect(0, 0, canvas.width, canvas.height);
        for (let j = 0; j < gameInfo.PlayerCount; j++){
            Game.drawDangerArea(j, dangerAreaDrawTable[j],gameInfo)
        }
        Game.drawArrow(canvas.width/2.0,canvas.height/2.0,gameInfo.SpawnerArrow.x ,gameInfo.SpawnerArrow.y);
        for (let j = 0; j < gameInfo.Balls.length ; j++){
            this.drawBall(gameInfo.Balls[j],gameInfo);
        }
        for (let j = 0; j < gameInfo.Platforms.length; j++){
            Game.drawBar(j,gameInfo.Platforms[j],gameInfo)
        }
    },

    drawArrow: function(fromx, fromy, tox, toy){
    var headlen = 5;   // length of head in pixels
    var angle = Math.atan2(toy-fromy,tox-fromx);
    ctx.beginPath();
        ctx.strokeStyle = '#999900';
        ctx.lineWidth = 2;
        ctx.moveTo(fromx, fromy);
        ctx.lineTo(tox, toy);
        ctx.lineTo(tox-headlen*Math.cos(angle-Math.PI/6),toy-headlen*Math.sin(angle-Math.PI/6));
        ctx.moveTo(tox, toy);
     ctx.lineTo(tox-headlen*Math.cos(angle+Math.PI/6),toy-headlen*Math.sin(angle+Math.PI/6));
     ctx.closePath();
     ctx.stroke();
     },

    drawBar: function (index, plat,gameInfo) {
        if (gameInfo.Lifes[index] === 0){
            return;
        }
        ctx.beginPath();
        ctx.fillStyle = ColorTable[index];
        ctx.fillRect(plat.x-(plat.w/2.0), plat.y-(plat.h/2.0), plat.w, plat.h);
        ctx.closePath();
        ctx.stroke();
    },

    drawDangerArea: function(index, drawIndex, gameInfo){
        if (gameInfo.Lifes[drawIndex] === 0){
            return;
        }
        ctx.beginPath();
        ctx.fillStyle = ColorTable[drawIndex];
        switch (index) {
            case 0:
                ctx.fillRect(0,canvas.height-gameInfo.DangerZoneSize,canvas.width,gameInfo.DangerZoneSize);
                break;
            case 1:
                ctx.fillRect(0,0,canvas.width,gameInfo.DangerZoneSize);
                break;
            case 2:
                ctx.fillRect(0,0,gameInfo.DangerZoneSize,canvas.height);
                break;
            case 3:
                ctx.fillRect(canvas.width-gameInfo.DangerZoneSize,0,gameInfo.DangerZoneSize,canvas.height);
                break;
        }
        ctx.closePath();
        ctx.stroke();
    },

    drawBall: function (pos,gameInfo) {
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.fillStyle = "black";
        ctx.arc(pos.x, pos.y, gameInfo.BallRadius, 0, 2 * Math.PI);
        ctx.closePath();
        ctx.fill();
    },
};

document.addEventListener('keydown', function(event) {
    switch (event.keyCode) {
        case 39: // Right key press
            movement.right = true;
            break;
        case 37: // Left key press
            movement.left = true;
            break;
        }
});
document.addEventListener('keyup', function(event) {
    switch (event.keyCode) {
        case 39: // Right key press
            movement.right = false;
            break;
        case 37: // Left key press
            movement.left = false;
            break;
    }
});