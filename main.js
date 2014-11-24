window.onload = function () {
  var answer;
  var offer = undefined;

  var getUrlParameter = function(sParam) {
    var sPageURL = window.location.search.substring(1);
    var sURLVariables = sPageURL.split('&');
    for (var i = 0; i < sURLVariables.length; i++)
    {
      var sParameterName = sURLVariables[i].split('=');
      if (sParameterName[0] == sParam)
      {
        return sParameterName[1];
      }
    }
    return undefined;
  };

  var cb = new Chatterbox();
  cb.onAddStream = function(stream) {
    trace('Presenting remote stream.');
    var element = document.querySelector("#remote");
    element.src = URL.createObjectURL(stream);
  };
  cb.onStreamAvailable = function(stream) {
    trace('Presenting local stream.');
    var element = document.querySelector("#local");
    element.src = URL.createObjectURL(stream);
  };

};
