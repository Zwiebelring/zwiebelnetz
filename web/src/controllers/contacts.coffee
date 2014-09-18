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
SecSocNet.ContactsController = Ember.ArrayController.extend
  search: ''
  allContacts: null

  content: (->
    @get('allContacts')
  ).property('allContacts')

  filterContacts: ->
    if @get('search') == ''
      @set 'content', @get('allContacts')
    else
      regexp = new RegExp(@get('search'),"i")
      @set 'content', @get('allContacts').filter (item, index, self) ->
        item.get('alias').match(regexp)

  searchPostsObserver: (->
    Ember.run.debounce(@,@filterContacts,500)
  ).observes('search')


  actions:
    followContact: ->
      onion = @get('contactOnion')
      alias = @get('contactAlias')
      return unless onion.trim()
      return unless alias.trim()
      onion = @store.createRecord 'onion',
        onion: onion
      onion.save().then( =>
        contact = @store.createRecord 'contact',
          onion: onion
          alias: alias
          status: 4 # following
        contact.save().then( =>
          @set 'contactOnion', ''
          @set 'contactAlias', ''
          @set 'requestMessage', ''
          @sync()
        ,->
          alert("Could not add contact, is the alias alright?")
        )
      ,->
        alert("Could not add onion, is the onion alright?")
      )
    createContactRequest: ->
      onion = @get('contactOnion')
      alias = @get('contactAlias')
      rmsg = @get('requestMessage')
      console.log(rmsg)
      return unless onion.trim()
      return unless alias.trim()
      return unless rmsg.trim()
      onion = @store.createRecord 'onion',
        onion: onion
      onion.save().then( =>
        console.log(rmsg)
        contact = @store.createRecord 'contact',
          onion: onion
          alias: alias
          request_message: rmsg
          status: 2 # pending
        console.log(contact)
        contact.save().then( =>
          @set 'contactOnion', ''
          @set 'contactAlias', ''
          @set 'requestMessage', ''
          @sync()
        ,->
          alert("Could not add contact, are alias and message alright?")
        )
      ,->
        alert("Could not add onion, is the onion alright?")
      )
  sync: ->
    user  = Ember.$.cookie('auth_user')
    token = Ember.$.cookie('auth_token')
    xhr = new XMLHttpRequest()
    xhr.open("post", "/sync", true);
    xhr.setRequestHeader('Auth-User', user)
    xhr.setRequestHeader('Auth-Token', token)
    xhr.send(null);
