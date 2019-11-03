import './helpers/bootstrap-helper';
import './scroll';
import './modal';

import $ from './helpers/jq-helpers';

$(document)
  .on('click', '.btn-group-toggle .btn', e => {
    $(e.target.closest('.btn-group-toggle')).$('label.btn.active').removeClass('active');
    $(e.target).addClass('active');
  })
  .on('click', '.dropdown-toggle', e => {
    const parent = e.target.parentElement;
    const dropdowns = $(parent).$('.dropdown-menu');
    if (parent.classList.contains('show')) {
      parent.classList.remove('show');
      dropdowns.removeClass('show');
    } else {
      parent.classList.add('show');
      dropdowns.addClass('show');
    }
  })
  .on('click', '.dropdown-item', e => {
    const dropdown = e.target.parentElement;
    const button = $(dropdown.parentElement).$('.dropdown-toggle');
    button.text(e.target.innerText);
    button.attr('data-value', e.target.dataset.value);
    $(dropdown).removeClass('show');
    $(dropdown.parentElement).removeClass('show');
  })
  .on('click', 'a[href*="event="], a[href*="e="]', e => {
    if (window.syna.stream._publishHashChange(e.target.href)) {
      e.preventDefault();
      return false;
    }
  });
