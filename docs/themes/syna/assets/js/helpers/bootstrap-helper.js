// Updated the script from https://stackoverflow.com/questions/43417452/animate-navbar-collapse-using-pure-js-css/43434017#43434017
import $ from './jq-helpers';

const toggle = document.querySelectorAll('.navbar-toggler');
const collapse = document.querySelectorAll('.navbar-collapse');
const dropdowns = document.querySelectorAll('.dropdown') || [];

function toggleMenu(node) {
  const menu = document.querySelector(node.dataset.target);
  menu.classList.toggle('in');
}

function closeMenus() {
  Array.from(dropdowns || []).forEach(node => {
    node.querySelector('.dropdown-toggle').classList.remove('dropdown-open');
    node.classList.remove('open');
  })
}

function closeMenusOnResize() {
  if (document.body.clientWidth >= 768) {
    closeMenus();
    Array.from(collapse || []).forEach(node => node.classList.remove('in'));
  }
}

function toggleDropdown() {
  if (document.body.clientWidth < 768) {
    var open = this.classList.contains('open');
    closeMenus();
    if (!open) {
      this.querySelector('.dropdown-toggle').classList.toggle('dropdown-open');
      this.classList.toggle('open');
    }
  }
}

window.addEventListener('resize', closeMenusOnResize, false);
Array.from(dropdowns || []).forEach(node => node.addEventListener('click', toggleDropdown))
Array.from(toggle || []).forEach(node => node.addEventListener('click', e => toggleMenu(node), false));
