// ==UserScript==
// @run-at		document-start
// @name         UserTest
// @namespace    http://tampermonkey.net/
// @version      0.1
// @description  try to take over the world!
// @author       wusp
// @match        https://ark.xiaohongshu.com/app-order/order/query
// @exclude		   这个和 iclude 配合使用，排除匹配到的网址，优先于 include
// @require      http://code.jquery.com/jquery-1.11.0.min.js
// @grant        unsafeWindow
// ==/UserScript==

(function () {
    'use strict';

    $(() => { 
    	function addXMLRequestCallback(callback){
            // 是一个劫持的函数
            var oldSend, i;
            if( XMLHttpRequest.callbacks ) {
                //   判断XMLHttpRequest对象下是否存在回调列表，存在就push一个回调的函数
                // we've already overridden send() so just add the callback
                XMLHttpRequest.callbacks.push( callback );
            } else {
                // create a callback queue
                XMLHttpRequest.callbacks = [callback];
                // 如果不存在则在xmlhttprequest函数下创建一个回调列表
                // store the native send()
                oldSend = XMLHttpRequest.prototype.send;
                // 获取旧xml的send函数，并对其进行劫持
                // override the native send()
                XMLHttpRequest.prototype.send = function(){
                    // process the callback queue
                    // the xhr instance is passed into each callback but seems pretty useless
                    // you can't tell what its destination is or call abort() without an error
                    // so only really good for logging that a request has happened
                    // I could be wrong, I hope so...
                    // EDIT: I suppose you could override the onreadystatechange handler though
                    for( i = 0; i < XMLHttpRequest.callbacks.length; i++ ) {
                        XMLHttpRequest.callbacks[i]( this );
                    }
                    // 循环回调xml内的回调函数
                    // call the native send()
                    oldSend.apply(this, arguments);
                //    由于我们获取了send函数的引用，并且复写了send函数，这样我们在调用原send的函数的时候，需要对其传入引用，而arguments是传入的参数
                }
            }
        }
        
        // e.g.
        addXMLRequestCallback( function( xhr ) {
                // 调用劫持函数，填入一个function的回调函数
                // 回调函数监听了对xhr调用了监听load状态，并且在触发的时候再次调用一个function，进行一些数据的劫持以及修改
                xhr.addEventListener("load", function(){
                if ( xhr.readyState == 4 && xhr.status == 200 ) {
                    console.log( xhr.response );
                }
            });
        
        });
    })
})();

