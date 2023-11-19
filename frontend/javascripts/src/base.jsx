import $ from 'jquery';

$(function () {
  $(".dropdown-container").on("mouseenter", function () {
    var t = $(this).data("menu-target");
    // alert(t);
    $(t).removeClass("hidden");
  }).on("mouseleave", function () {
    var t = $(this).data("menu-target");
    // alert(t)
    // alert("hide menu");
    $(t).addClass("hidden");

  });
})
