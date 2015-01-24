var tvcontrollers = angular.module('tvControllers', [] );

tvcontrollers.controller('HomeCtrl', [ '$scope', '$http', '$location',
								function($scope, $http, $location) {
									$scope.categories = [];
									//$scope.categories = [ {"CategoryName": "One", "videos": [{"Title": "Video Title"}]}];	

									$http.get('playlists').success(function(data) {
										for (var key in data) {
  										if (data.hasOwnProperty(key)) {
													$scope.categories.push({"CategoryName": key, "videos": data[key] });
  											}
										}

									});

									$scope.navigateToVideo = function(id) {
										$location.path('/video/' + id);
									};
								}]);

tvcontrollers.controller('CategoryDetailCtrl', [ '$scope', '$http',
								function($scope, $http) {
									$http.get('controllerDetailRoute').success(function(data){
										$scope.controllerInfo = data;
									});

								}]);

tvcontrollers.controller('VideoCtrl', [ '$scope', '$http', 
								function($scope, $http) {
									$http.get('video/').success(function(data) {
										$scope.video = data;
										
										angular.element(document).ready(function () {
													$scope.setupYoutubePlayer();
													console.log('Hello World');
    								});
									});
									
									$scope.setupYoutubePlayer = function() {
										// Load the IFrame Player API code asynchronously.
										var tag = document.createElement('script');
										tag.src = "https://www.youtube.com/player_api";
										var firstScriptTag = document.getElementsByTagName('script')[0];
										firstScriptTag.parentNode.insertBefore(tag, firstScriptTag);

										// Replace the 'ytplayer' element with an <iframe> and
										// YouTube player after the API code downloads.
										var player;
										function onYouTubePlayerAPIReady() {
											var id = document.getElementById('video_id').value;
											player = new YT.Player('ytplayer', {
												height: '390',
												width: '640',
												videoId: id
											});
										}	
									};
								}]);
