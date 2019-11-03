import $ from './helpers/jq-helpers';

const collapse = $('[data-toggle="collapse"]');
const addCollapse = $('[data-toggle="collapse"][data-add-collapse]');

addCollapse.$nodes.forEach(collapsible => {
  const target = $(collapsible.dataset.target);

  if (target && target[0].children.length) {
    const node = $(collapsible.dataset.addCollapse);
    node.append('<i class="fas fa-chevron-down text-primary"></i>');
  }
});

collapse.on('click', function (e) {
  if (e.target.tagName === 'A') {
    return
  }
  const target = $(this).attr('data-target');

  if ($(this).attr('aria-expanded') === 'true') {
    hideCollapse(this, target);
  } else {
    showCollapse(this, target);
  }
});

const hideCollapse = function (el, target) {
  $(el).attr('aria-expanded', 'false');
  $(el).addClass('collapsed');
  $(target).removeClass('show');
};

const showCollapse = function (el, target) {
  $(el).attr('aria-expanded', 'true');
  $(el).removeClass('collapsed');
  $(target).addClass('show');
};
