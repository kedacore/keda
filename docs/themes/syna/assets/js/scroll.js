import $ from './helpers/jq-helpers';

(function() {
  handleScroll()
  window.onscroll = handleScroll;
  $('.scroll-to-top').on('click', scrollToTop);
})();

function handleScroll() {
  if (window.scrollY > window.innerHeight / 2) {
    $('.scroll-to-top').removeClass('d-none');
  } else {
    $('.scroll-to-top').addClass('d-none');
  }

  const headers = $('.content-fragment h1, .content-fragment h2, .content-fragment h3, .content-fragment h4, .content-fragment h5, .content-fragment h6, .fragment');
  for (let i = headers.length - 1; i >= 0; i--) {
    const bounds = headers[i].getBoundingClientRect();
    if (bounds.top < 64) {
      $('.scroll-spy a:not(.default-active)').removeClass('active');
      $('.toc #TableOfContents li a').removeClass('active');
      if (headers[i].id) {
        $(`.toc #TableOfContents li a[href="${window.location.pathname}#${headers[i].id}"]`).addClass('active');
        $(`.scroll-spy a[href="${window.location.pathname}#${headers[i].id}"]`).addClass('active');
      }
      break;
    }
  }
}

function scrollToTop() {
  $.scrollTo(document.body, 0, 250)
}
