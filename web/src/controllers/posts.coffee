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
SecSocNet.PostsController = Ember.ArrayController.extend
  sortAscending: false
  sortProperties: ["postedAt"]
  ttl: [1,2,3,4,5]
  selectedTTL: 3
  selectedCircles: null
  allPosts: null
  search: ''

  content: (->
    @get('allPosts')
  ).property('allPosts')

  filterPosts: ->
    if @get('search') == ''
      @set 'content', @get('allPosts')
    else
      posts = @get('allPosts')
      authors = @get('search').match(/author:"([\w|\s|\.]*)"/i)
      if authors
        authors.shift()
        if "me" in authors
          @store.find('author',1).then (me) =>
            posts = posts.filter (item, index, self) -> item.get('author.onion') == me.get('onion')
        else
          #only onion addr support right now
          for author in authors
            regexp = new RegExp(author,"i")
            posts = posts.filter (item, index, self) -> item.get('author.onion').match(regexp)
        @set 'content', posts
      circles = @get('search').match(/circle:"([\w|\s|\.]*)"/i)
      if circles
        circles.shift()
        for circle in circles
          regexp = new RegExp(circle,"i")
          posts = posts.filter (item, index, self) ->
            item.get('circles').any (circ) -> circ.get('name').match(regexp)
        @set 'content', posts

  searchPostsObserver: (->
    Ember.run.debounce(@,@filterPosts,500)
  ).observes('search')


  actions:
    addCircle: (id) ->
    removeCircle: (id) ->
    createPost: ->
      message = @get('message')
      return unless message.trim()

      originator = @store.find('originator',1)
      author = @store.find('author',1)
      Ember.RSVP.all([originator, author]).then =>
        post = @store.createRecord 'post',
          originator: originator
          author: author
          message: message
          ttl: @get("selectedTTL")

        circle_ids = $('.select2-container').select2("val")
        @store.find('circle', {ids: circle_ids}).then (circles) =>
          post.get('circles').pushObjects circles
          post.save().then( =>
            @set "message", ""
          , => alert("Something went wrong"))
