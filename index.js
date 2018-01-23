'use strict';
const Alexa = require('alexa-sdk');
const request = require('request');

exports.handler = function(event, context, callback) {
    let alexa = Alexa.handler(event, context);
    alexa.resources = languageStrings;
    alexa.registerHandlers(handlers,firstHandlers,secondHandlers, thirdHandlers);
    alexa.execute();
};

const jsonURL = "https://s3-ap-northeast-1.amazonaws.com/rebuild.bucket/rebuild.json";

var STATUS = {
    FIRSTMODE: '_FIRSTMODE',
    SECONDMODE: '_SECONDMODE',
    THIRDMODE: '_THIRDMODE'
};

const languageStrings = {
  'ja-JP': {
    'translation': {
      'CANCEL' : "キャンセルしました。また、遊んで下さいね",
      'FIRST' : "Rebuildの最終回をお聞きしますか？",
      'SECOND' : "ポストしました。"
    }
  }
};

var handlers = {
    'Unhandled': function () {
        this.handler.state = STATUS.FIRSTMODE;
        this.emitWithState("First", false);
    },
    "AMAZON.CancelIntent": function() {
        this.response.audioPlayerStop();
        this.emit(':responseReady');
        this.emit(':tell', this.t("CANCEL"));
    },
    "AMAZON.StopIntent": function() {
        this.response.audioPlayerStop();
        this.emit(':responseReady');
    },
    "AMAZON.PauseIntent" : function () {
        this.response.audioPlayerStop();
        this.emit(':responseReady');
    }
};

var firstHandlers = Alexa.CreateStateHandler(STATUS.FIRSTMODE, {
    'First': function () {
        this.handler.state = STATUS.SECONDMODE;
        this.emit(':ask', this.t("FIRST")); // askは会話が続く
    }
});

var secondHandlers = Alexa.CreateStateHandler(STATUS.SECONDMODE, {
    'Second': function () {
        var usersAnswer = this.event.request.intent.slots.Answer.value;
        var THIS = this;
        if (usersAnswer == "はい" || usersAnswer == "イェス") {
            request(jsonURL, function (error, response, body) {
                if (response.statusCode != 200) {
                    THIS.emit(':tell', "リクエストに失敗しました");
                    return
                }
                let json = JSON.parse(body);
                let items = json["items"];
                let item = items[0];
                let title = item["title"];
                let url = item["url"];
                THIS.response.speak(title).audioPlayerPlay('REPLACE_ALL', url, url, null, 0);
                THIS.emit(':responseReady');    
            });
        } else {
            request(jsonURL, function (error, response, body) {
                if (response.statusCode != 200) {
                    THIS.emit(':tell', "リクエストに失敗しました");
                    return
                }
                let json = JSON.parse(body);
                let items = json["items"];
                var title = items[0]["title"];
                var count = (title).replace(/[^0-9]/g, '');
                THIS.handler.state = STATUS.THIRDMODE;
                THIS.emit(':ask', THIS.t("合計"+count+"回あります、何回目を再生しますか？"));
            });
        }
    }
});

var thirdHandlers = Alexa.CreateStateHandler(STATUS.THIRDMODE, {
    'Third': function () {
        var Number = this.event.request.intent.slots.Number.value;
        var THIS = this;
        request(jsonURL, function (error, response, body) {
            if (response.statusCode != 200) {
                THIS.emit(':tell', "リクエストに失敗しました");
                return
            }
            var json = JSON.parse(body);
            var items = json["items"];
            var played = false;
            items.forEach(function(item){
                var title = item["title"];
                var url = item["url"];
                var count = (title).replace(/[^0-9]/g, '');
                if (count == Number) {
                    played = true;
                    THIS.response.speak(title).audioPlayerPlay('REPLACE_ALL', url, url, null, 0);
                    THIS.emit(':responseReady');
                }
            })
            if (played == false) {
                THIS.emit(':ask', THIS.t("もう一度お願いします、何回目を再生しますか？")); // askは会話が続く
            }
        });
    }
});
