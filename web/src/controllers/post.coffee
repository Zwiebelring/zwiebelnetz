#
# Copyright (c) 2014
#   Dario Brandes
#   Thies Johannsen
#   Paul Kröger
#   Sergej Mann
#   Roman Naumann
#   Sebastian Thobe
# All rights reserved.
#
# Redistribution and use in source and binary forms, with or without modification,
# are permitted provided that the following conditions are met:
#
# 1. Redistributions of source code must retain the above copyright notice, this
#    list of conditions and the following disclaimer.
#
# 2. Redistributions in binary form must reproduce the above copyright notice,
#    this list of conditions and the following disclaimer in the documentation
#    and/or other materials provided with the distribution.
#
# THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
# ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
# WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
# DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
# ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
# (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
# LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
# ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
# (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
# SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

#  -*-  indent-tabs-mode: nil; c-basic-offset: 4; tab-width: 4 -*-
SecSocNet.PostController = Ember.ObjectController.extend
  commentsVisible: false
  publicComment: true

  showComments: (->
    @get('commentCount') > 0 && @get('commentsVisibility')
  ).property('commentCount','commentsVisibility')

  hasComments: (->
    @get('commentCount') > 0
  ).property('commentCount')

  commentButtonLabel: (->
    label = ""
    label += if @get('commentsVisibility') then "Hide " else "Show "
    label += "Comments "
    label += "(" + @get('commentCount') + ")"
    #label += if @get('commentCount') > 1 then " comments" else " comment"
  ).property('commentCount','commentsVisibility')

  commentCount: (->
    @get('comments').get('length')
  ).property('comments.@each')

  actions:
    createComment: ->
      parent = @get('model')
      message = @get('commentMessage')
      return unless message.trim()
      @store.find('originator',1).then (originator) =>
        @store.find('author',1).then (author) =>
          comment = @store.createRecord 'comment',
            author: author
            originator: originator
            parent: parent
            message: message
            ttl: if @get('publicComment') then 2 else 1
          parent.get('comments').then (comments) =>
            comments.pushObject(comment)
            comment.save().then( =>
              @set 'commentMessage', ""
              parent.reload()
            , => alert("Something went wrong"))
    showCommentsSection: ->
      @set 'commentsVisible', true
    hideCommentsSection: ->
      @set 'commentsVisible', false
    deletePost: ->
      post = @get('model')
      post.get('comments').then (_comments) =>
        _comments.forEach (_comment) =>
          _comment.deleteRecord()
          _comment.save()
      post.deleteRecord()
      post.save()
    togglePublicComment: ->
      @set 'publicComment', !@get('publicComment')
