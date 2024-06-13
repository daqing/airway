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

  $(".goto-link").on("click", function () {
    var href = $(this).data("href");
    var id = $(this).data("id");

    var url = href + id;

    window.location.href = url;
  });
})
