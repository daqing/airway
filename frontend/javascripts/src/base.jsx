import $ from 'jquery';

window.$ = $;

$(function () {
  $(".dropdown-container").on("mouseenter", function () {
    var t = $(this).data("menu-target");
    $(t).removeClass("hidden");
  }).on("mouseleave", function () {
    var t = $(this).data("menu-target");
    $(t).addClass("hidden");
  });
})
