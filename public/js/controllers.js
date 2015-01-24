var tvcontrollers = angular.module('tvControllers', [] );

tvcontrollers.controller('HomeCtrl', [ '$scope', '$http',
								function($scope, $http) {
									$http.get('route1').success(function(data) {
										$scope.categories = data;
									});
								
									$scope.categories = [];	
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
