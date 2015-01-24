var tvcontrollers = angular.module('tvControllers', [] );

tvcontrollers.controller('HomeCtrl', [ '$scope', '$http',
								function($scope, $http) {
									$scope.categories = [];
									//$scope.categories = [ {"CategoryName": "One", "videos": [{"Title": "Video Title"}]}];	

									$http.get('playlists').success(function(data) {
										for (var key in data) {
  										if (p.hasOwnProperty(key)) {
													$scope.categories.push({"CategoryName": key, "videos": data[key] });
  											}
										}

									//	console.log($scope.categories.Keys());
									});
								}]);

tvcontrollers.controller('CategoryDetailCtrl', [ '$scope', '$http',
								function($scope, $http) {
									$http.get('controllerDetailRoute').success(function(data){
										$scope.controllerInfo = data;
									});

								}]);

tvcontrollers.controller('VideoCtrl', [ '$scope', '$http', 
								function($scope, $http) {
									$http.get('videoDetailRoute').success(function(data) {
										$scope.videoDetails = data;
									});
									
								}]);
