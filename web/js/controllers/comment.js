// Generated by CoffeeScript 1.7.1
(function() {
  SecSocNet.CommentController = Ember.ObjectController.extend({
    actions: {
      deleteComment: function() {
        var comment;
        comment = this.get('model');
        comment.deleteRecord();
        return comment.save();
      }
    }
  });

}).call(this);