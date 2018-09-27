var canvas = document.getElementById("game");
var ctx = canvas.getContext('2d');
var w = canvas.width; // variable globale w pour la largeur du canvas
var h = canvas.height; // variable globale h pour la hauteur du canvas
var x = 0;  // variable globale x pour la position de la souris
var y = 0; // variable globale y pour la position de la souris

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

var nameInput = document.getElementById("nameInput");
var output = document.getElementById("output");
var loginButton = document.getElementById("loginButton");
var socket = new WebSocket("ws://localhost:8080/ws");
// socket.binaryType = 'arraybuffer';

socket.onopen = function () {
    console.log("Created websocket connection");
    var session = {};
    session.Message = "Session";
    session.Data = {};
    session.Data.Cookie = getSessionCookie();
    socket.send(JSON.stringify(session));
};

socket.onmessage = function (msg){
    console.log(msg);
    var json = JSON.parse(msg.data);
    if(json.hasOwnProperty('Message')){
        switch (json.Message) {
            case "Player":
                receivedPlayerInfo(json);
                break;
            case "Error":
                receivedError(json);
                break;
        }
    }
    // switch (json.message) {
    //     case "newplayer":
    //         newPlayerMessage(json);
    //         break;
    // }
    output.innerHTML +="Server: " + JSON.stringify(json) + "\n";
};



function receivedError(json){
    window.alert(json.Data.ErrorMsg);
}

function receivedPlayerInfo(json){
    console.log("Received player info");
    nameInput.value = json.Data.Name;
    nameInput.disabled = true;
    loginButton.disabled = true;
    var expires = "";
    var date = new Date();
    date.setDate(date.getDate() + 1);
    expires = "expires=" + date.toUTCString()
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


function login(){
    console.log("Login in");
    var msg = {};
    msg.Message = "Player";
    msg.Data = {};
    msg.Data.Name = nameInput.value;
    socket.send(JSON.stringify(msg));
}



var Game = {
    init: function () {
        for (var i = 0; i < 10; i++)
            for (var j = 0; j < 15; j++)
                Game.gameArea.push(new this.bar(j * 60, i * 30, 40, 5, Math.floor((Math.random() * 3) + 1)));

        this.barJoueur = new this.bar(x, y, 100, 10);
        this.ball = new this.balle(100, 300, 10, 8, 8);

        canvas.onmousemove = function (e) {
            x = e.pageX - this.offsetLeft;
            y = e.pageY - this.offsetTop;
            Game.barJoueur.x = e.pageX - this.offsetLeft;
            Game.barJoueur.y = e.pageY - this.offsetTop;
        };


        this.render();
    },
    bar: function (x, y, w, h, life) {
        this.x = x;
        this.y = y;
        this.height = h;
        this.width = w;
        this.life = life;
        if (typeof this.initialized === "undefined") {
    
            this.x2 = function () {
                return this.x + this.width;
            };

            this.y2 = function () {
                return this.y + this.height;
            };
            this.initialized = true;
        }
    },
    balle: function (x, y, r, velx, vely) {
        this.x = x;
        this.y = y;
        this.r = r;
        this.velx = velx;
        this.vely = vely;
        if (typeof this.initialized === "undefined") {
            this.x1 = function () {
                return this.x - this.r;
            };

            this.y1 = function () {
                return this.y - this.r;
            };
            this.x2 = function () {
                return this.x + this.r;
            };

            this.y2 = function () {
                return this.y + this.r;
            };
            this.initialized = true;
        }
    },
    drawBar: function (bar) {
        ctx.beginPath();
        ctx.fillRect(bar.x, bar.y, bar.width, bar.height);
        ctx.closePath();
        ctx.stroke();
    },
    gameArea: [],
    draw: function () {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        //ctx.clearRect(ball.x-ball.r-1, ball.y-ball.r-1, ball.r*2+1, ball.r*2+1);
        this.collisionArea();
        this.mouvement();

        this.drawBar(this.barJoueur);

        if (this.isIntersect(this.barJoueur, this.ball) === true)
            console.log(true);

        this.drawAllbar(this.gameArea);
        this.drawBall();
    },
    render: function () {
        Game.draw();
        requestAnimFrame(Game.render);
    },
    collisionArea: function () {
        //Bords verticaux
        if (this.ball.x + this.ball.r >= w || this.ball.x - this.ball.r <= 0)
            this.ball.velx *= -1;

        //Bords horizontaux
        if (this.ball.y - this.ball.r <= 0 || this.ball.y + this.ball.r >= h)
            this.ball.vely *= -1;
    },
    mouvement: function () {
        this.ball.x += this.ball.velx;
        this.ball.y += this.ball.vely;
    },
    drawAllbar: function (gameArea) {
        ctx.beginPath();
        for (var i = 0; i < gameArea.length; i++) {
            if (gameArea[i]) {
                if (gameArea[i].life === 3)
                    ctx.fillStyle = '#00ff3f';
                else if (gameArea[i].life === 2)
                    ctx.fillStyle = '#ffe900';
                else
                    ctx.fillStyle = '#ff0000';

                ctx.fillRect(gameArea[i].x, gameArea[i].y, gameArea[i].width, gameArea[i].height);
                this.isIntersect(gameArea[i], this.ball, i);
            }
        }
        ctx.closePath();
        ctx.stroke();
    },
    drawBall: function () {
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.arc(this.ball.x, this.ball.y, this.ball.r, 0, 2 * Math.PI);
        ctx.closePath();
        ctx.fill();
    },
    isIntersect: function (bar, ball, i) {
        if (bar.x >= ball.x2() + 2 || bar.x2() <= ball.x1() - 2 || bar.y >= ball.y2() + 2 || bar.y2() <= ball.y1() - 2)
            return false;

        //colision de la barre
        if (!i) {
            if (!(bar.x < ball.x1() - 15 && bar.x2() > ball.x2() + 15))
                ball.velx *= -1;
            ball.vely *= -1;
        }
        else {

            if (ball.x < bar.x && ball.y + ball.r > bar.y && ball.y - ball.r < bar.y2()
			    || ball.x > bar.x2() && ball.y + ball.r > bar.y && ball.y - ball.r < bar.y2())
                ball.velx *= -1;
            else
                ball.vely *= -1;


            if (this.gameArea[i].life === 1)
                delete this.gameArea[i];
            else
                this.gameArea[i].life--;

        }

        return true;
    },
    distanceMilieu: function (x, x2, y, y2) {

        return ((x - x2) ^ 2 + (y - y2) ^ 2) ^ 0.5;
    },
    unset: function (array, value) {
        array.splice(array.indexOf(value), 1);
    }
};
Game.init();