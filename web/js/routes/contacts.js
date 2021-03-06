// Generated by CoffeeScript 1.7.1
(function() {
  SecSocNet.ContactsRoute = Ember.Route.extend({
    setupController: function(controller) {
      controller.set('allContacts', this.get('store').find('contact'));
      return setInterval(this.reload.bind(this), 5000);
    },
    renderTemplate: function() {
      return this.render('contacts', {
        into: 'application'
      });
    },
    reload: (function() {
      var token, user, xhr;
      user = Ember.$.cookie('auth_user');
      token = Ember.$.cookie('auth_token');
      xhr = new XMLHttpRequest();
      xhr.open("get", "/pending_contacts", true);
      xhr.setRequestHeader('Auth-User', user);
      xhr.setRequestHeader('Auth-Token', token);
      xhr.send(null);
      return xhr.onreadystatechange = ((function(_this) {
        return function() {
          var resp;
          if (xhr.readyState === 4) {
            resp = xhr.getResponseHeader("pending");
            if (resp === "true") {
              console.log("Reloading contacts ...");
              return _this.store.find('contact');
            }
          }
        };
      })(this));
    })
  });

}).call(this);
