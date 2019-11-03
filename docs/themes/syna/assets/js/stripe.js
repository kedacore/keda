import $ from './helpers/jq-helpers';
import Validator from 'form-validator-simple';
import parsePrice from 'parse-price';

function initFormValidation(form, onSuccess = () => false) {
  new Validator({
    errorTemplate: '<span class="help-block form-error">%s</span>',
    onFormValidate: (isFormValid, form) => {
      form.querySelector('button').disabled = !isFormValid;
    },
    onError: function(e, form) {
      form.querySelector('.generic-error').removeClass('d-none');
    },
    onSuccess,
    form,
  });
}

function onSubmit(configId, form, stripe, card) {
  return function(e) {
    e.preventDefault();

    const config = window.syna.api.get('stripe', configId);
    const button = form.$('button.submit-btn');
    button.attr('disabled', true).addClass('disabled');
    form.$('#generic-error').addClass('d-none');
    form.$('#generic-success').addClass('hidden');
    form.removeClass('success').removeClass('error');
    // Stripe requires creating a token for user data to avoid sending data to other server
    stripe.createToken(card).then(result => {
      if (result.error) {
        $('.invalid-feedback').text(result.error.message);
        button.removeAttr('disabled').removeClass('disabled');
        card.clear();
        card.focus();
        form.addClass('error');
      } else {
        const action = form.attr('action');

        // Parse the form data and calculate the price based on whether the form
        // had single value, custom value or multiple values
        const formData = JSON.parse(form.serialize(true));
        let price = formData.price_text;
        const serializedForm = {
          email: formData.email,
          stripeToken: result.token.id,
          metadata: Object.assign(formData, {
            product: config.product,
            description: config.description,
            from: window.location.href,
          }),
        };

        if (formData.custom_value === "true") {
          price = formData.custom_price_text;
          serializedForm.currency = form.$('[data-input=currency]').attr('data-value');
        } else {
          serializedForm.currency = form.$('input[name=price_text]:checked').attr('data-currency');
        }
        serializedForm.metadata.price_text = price;
        serializedForm.price = parsePrice(price) * (currencies[serializedForm.currency.toUpperCase()] || 1);

        $.post(action, JSON.stringify(serializedForm))
          .then(() => {
            button.removeAttr('disabled').removeClass('disabled');
            form.$('#generic-success').removeClass('hidden');
            form.addClass('success');
          })
          .catch(() => {
            button.removeAttr('disabled').removeClass('disabled');
            form.$('#generic-error').removeClass('d-none');
            form.addClass('error');
            card.clear();
            card.focus();
          });
      }
    });
  }
}

const stripeFragments = window.syna.api.getScope('stripe');
Object.keys(stripeFragments).forEach(key => {
  const config = stripeFragments[key];
  const stripe = Stripe(config.token);
  const elements = stripe.elements();
  const card = elements.create('card', config.options);
  card.mount(`#payment-form-${config.form} #card-element`);
  card.addEventListener('change', e => {
    const displayError = $('.invalid-feedback');
    if (event.error) {
      displayError.text(event.error.message);
    } else {
      displayError.text('');
    }
  });

  const form = $(`#payment-form-${config.form}`);
  initFormValidation(form[0], onSubmit(key, form, stripe, card));

  if (form.$('input[name=custom_price_text]').length > 0) {
    form.$('[data-render="price-value"]').text(form.$('input[name=custom_price_text]').val() + form.$('[data-input=currency]').text())
  }
  const choices = $(`#payment-form-${config.form} input[name=price_text]`);
  if (choices.length > 0) {
    choices.$nodes[0].setAttribute('checked', true);
    choices.$nodes[0].parentElement.classList.add('active');
    form.$('[data-render="price-value"]').text(choices.$nodes[0].value)
  }

  form.$('[data-action="toggle-price-change"]').on('click', (() => {
    let isEditable = false;
    return () => {
      if (isEditable) {
        form.$('.price-display').removeClass('hidden');
        form.$('.price-input').addClass('hidden');
        form.$('input[name=custom_value]').val('false');
        form.$('[data-render="price-value"]').text(form.$('input[name=price_text][checked]').val());
      } else {
        form.$('.price-display').addClass('hidden');
        form.$('.price-input').removeClass('hidden');
        form.$('input[name=custom_value]').val('true');
        form.$('[data-render="price-value"]').text(form.$('input[name=custom_price_text]').val() + form.$('[data-input=currency]').text());
      }
      isEditable = !isEditable;
    }
  })());

  form.on('input', 'input[name=price_text]', e => {
    const price = parseInt(e.target.value.match(/\w+/g).reduce((tmp, match) => tmp + match, ''), 10)
    form.$('input[name=price]').val(price);
    form.$('[data-render="price-value"]').text(e.target.value);
  });
  form.on('input', 'input[name=custom_price_text]', e => {
    form.$('[data-render="price-value"]').text(e.target.value + form.$('[data-input=currency]').text());
  })

  form.$('#generic-success [data-action="return-form"]').on('click', () => {
    form.$('#generic-success').addClass('hidden');
    form.removeClass('success');
  });
});

window.syna.stream.subscribe('pricing:change', function({ product, description, price, currency }) {
  updateStripeFragments(product, description, price, currency);
});

function updateStripeFragments(product, description, price, currency) {
  window.syna.api.toArray('stripe').forEach(config => {
    const form = $(`#payment-form-${config.form}`);

    config.description = description
    config.product = product;

    if (product) {
      $('[data-render="product"]').html(
        window.syna.api.renderTemplate(
          $(`#stripe-product-template-${config.form}`).html(),
          { product }
        ),
      );
    }

    if (price) {
      const priceTemplate = $(`#stripe-price-template-${config.form}`).html();
      const data = { price, currency };

      if (form.$('[data-render=price]').length > 0) {
        form.$('[data-render=price]').html(window.syna.api.renderTemplate(priceTemplate, data));
      }

      if (form.$('[data-value=price]').length > 0) {
        form.$('[data-value=price]').val(price);
      }

      form.$('[data-render="price-value"]').text(price);

      setTimeout(() => {
        const choices = $(`#payment-form-${config.form} input[name=price_text]`);
        if (choices.length > 0) {
          choices.$nodes[0].setAttribute('checked', true);
          choices.$nodes[0].parentElement.classList.add('active');
        }
      }, 0);
    }

    if (currency) {
      form.$('input[name=currency]').text(currency);
    }

    form.$('input[name=email]')[0].focus();
    // TODO: REVISIT: Remove the following line whenever firefox fixes center on focus
    form[0].scrollIntoView({behavior: "instant", block: "center"});
  });
}

const currencies = {
  AED: 100,
  AFN: 100,
  ALL: 100,
  AMD: 100,
  ANG: 100,
  AOA: 100,
  ARS: 100,
  AUD: 100,
  AWG: 100,
  AZN: 100,
  BAM: 100,
  BBD: 100,
  BDT: 100,
  BGN: 100,
  BHD: 1000,
  BIF: 1,
  BMD: 100,
  BND: 100,
  BOB: 100,
  BRL: 100,
  BSD: 100,
  BTN: 100,
  BWP: 100,
  BYN: 100,
  BZD: 100,
  CAD: 100,
  CDF: 100,
  CHF: 100,
  CLP: 1,
  CNY: 100,
  COP: 100,
  CRC: 100,
  CUC: 100,
  CUP: 100,
  CVE: 100,
  CZK: 100,
  DJF: 1,
  DKK: 100,
  DOP: 100,
  DZD: 100,
  EGP: 100,
  ERN: 100,
  ETB: 100,
  EUR: 100,
  FJD: 100,
  FKP: 100,
  GBP: 100,
  GEL: 100,
  GGP: 100,
  GHS: 100,
  GIP: 100,
  GMD: 100,
  GNF: 1,
  GTQ: 100,
  GYD: 100,
  HKD: 100,
  HNL: 100,
  HRK: 100,
  HTG: 100,
  HUF: 100,
  IDR: 100,
  ILS: 100,
  IMP: 100,
  INR: 100,
  IQD: 1000,
  IRR: 100,
  ISK: 100,
  JEP: 100,
  JMD: 100,
  JOD: 100,
  JPY: 1,
  KES: 100,
  KGS: 100,
  KHR: 100,
  KID: 100,
  KMF: 1,
  KPW: 100,
  KRW: 1,
  KWD: 1000,
  KYD: 100,
  KZT: 100,
  LAK: 100,
  LBP: 100,
  LKR: 100,
  LRD: 100,
  LSL: 100,
  LYD: 1000,
  MAD: 100,
  MDL: 100,
  MGA: 1,
  MKD: 100,
  MMK: 100,
  MNT: 100,
  MOP: 100,
  MRU: 5,
  MUR: 100,
  MVR: 100,
  MWK: 100,
  MXN: 100,
  MYR: 100,
  MZN: 100,
  NAD: 100,
  NGN: 100,
  NIO: 100,
  NOK: 100,
  NPR: 100,
  NZD: 100,
  OMR: 1000,
  PAB: 100,
  PEN: 100,
  PGK: 100,
  PHP: 100,
  PKR: 100,
  PLN: 100,
  PRB: 100,
  PYG: 1,
  QAR: 100,
  RON: 100,
  RSD: 100,
  RUB: 100,
  RWF: 1,
  SAR: 100,
  SBD: 100,
  SCR: 100,
  SDG: 100,
  SEK: 100,
  SGD: 100,
  SHP: 100,
  SLL: 100,
  SLS: 100,
  SOS: 100,
  SRD: 100,
  SSP: 100,
  STN: 100,
  SYP: 100,
  SZL: 100,
  THB: 100,
  TJS: 100,
  TMT: 100,
  TND: 1000,
  TOP: 100,
  TRY: 100,
  TTD: 100,
  TVD: 100,
  TWD: 100,
  TZS: 100,
  UAH: 100,
  UGX: 1,
  USD: 100,
  UYU: 100,
  UZS: 100,
  VES: 100,
  VND: 1,
  VUV: 1,
  WST: 100,
  XAF: 1,
  XCD: 100,
  XOF: 1,
  XPF: 1,
  YER: 100,
  ZAR: 100,
  ZMW: 100,
  ZWB: 100
};
