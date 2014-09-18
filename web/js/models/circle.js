// Generated by CoffeeScript 1.7.1
(function() {
  SecSocNet.Circle = DS.Model.extend({
    name: DS.attr('string'),
    creator: DS.attr('number'),
    contacts: DS.hasMany('contact', {
      async: true
    }),
    posts: DS.hasMany('post', {
      async: true
    }),
    profiles: DS.hasMany('profile', {
      async: true
    }),
    isPublicCircle: (function() {
      return this.get("name") === "Public";
    }).property("name")
  });

}).call(this);