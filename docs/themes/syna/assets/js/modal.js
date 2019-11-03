import $ from './helpers/jq-helpers';

const modal = $('.modal');
const dialog = $('.modal .modal-dialog');

function closeDialog() {
  $('body').removeClass('modal-open');
  modal.removeClass('show');
}

$('[data-dismiss="modal"]').on('click', closeDialog);

modal.on('click', e => { 
  if (!dialog[0].contains(e.target)) {
    closeDialog()
  }
});

(window.syna || (window.syna = {})).showModal = function({ title, subtitle, image, content, labels, size = '' }) {
  $('body').addClass('modal-open');
  modal.addClass('show');
  dialog.$('.title').html(title || '');
  dialog.$('.subtitle').html(subtitle || '');
  dialog.$('.badge-container').html(labels || '');
  dialog.$('img')[0].src = image;
  if (content) {
    dialog.$('.modal-body .content').html(content);
    dialog.$('.modal-body .content').removeClass('hidden');
  } else {
    dialog.$('.modal-body .content').addClass('hidden');
  }
  dialog.removeClass('md').removeClass('lg').addClass(size);
}
