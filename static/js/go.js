$(document).ready(function() {
  $('.in').on('input',function() {
    var allvals = $('.in').map(function() { 
        return this.value; 
    }).get().join('/');
    $('.concat-usage').val( allvals );
  });
});