'use strict';

(function (window, document) {


// TODO: multiple request together
// do we cancel prev or bounce the request
// e.g. search, search, search
//           <-[cacel]<-[cancel]
// e.g. list,  list,  list
//          <-[wait]<-[wait]
var api = {
   user: {
      checkLogin: function () {
         return Promise.resolve(true);
      }, // checkLogin
      test: 0
   }, // user
   project: {
      getList: function () {
         return Promise.resolve(['test1', 'test2', 'test3']);
      }, // getList
      getDirectoryContents: function (project, path) {
         return Promise.resolve([
            { name: 'next/' },
            { name: 'pcakge.json' },
            { name: 'README.md' }
         ]);
      }, // getDirectoryContents
      getFileContents: function (project, path) {
         return Promise.resolve({
            binary: false,
            data: 'This is a test readme file.'
         });
      }, // getFileContents
      search: function (query, n) {
         return Promise.resolve({
            matchRegexp: '[Tt]his is',
            items: [
               { path: '/test1/README.md', matches: [
                  { L: 1, T: 'This is a test readme file.' }
               ] }
            ]
         });
      } // search
   } // project
};

if (!window.Flame) window.Flame = {};
window.Flame.api = api;

})(window, document);
