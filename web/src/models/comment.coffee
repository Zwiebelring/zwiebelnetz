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

SecSocNet.Comment = DS.Model.extend
  message:    DS.attr 'string'
  createdAt:  DS.attr 'string',
    defaultValue: ->
      new Date().toISOString()
  postedAt:  DS.attr 'string',
    defaultValue: ->
      new Date().toISOString()
  updatedAt:  DS.attr 'string',
    defaultValue: ->
      new Date().toISOString()
  ttl: DS.attr 'number'
  profilePictureId: DS.attr 'number'

  author:             DS.belongsTo('author', async: true)
  originator:         DS.belongsTo('originator', async: true)
  post:               DS.belongsTo('post', async: true)
  isRemotePublished:  DS.attr 'boolean'

  profilePicture: "img/no_profile_picture.jpg"
  profilePictureCall: (->
    if @get('profilePictureId') > 0              
      profile = @store.find('profile', @get('profilePictureId')).then (_profile) =>          
        @set 'profilePicture',  "data:image/jpg;base64," + _profile.get('value')
    else
      @set 'profilePicture', "img/no_profile_picture.jpg"
  ).property('profilePictureId')


  contactAlias: ""
  contact: (->
    @get('author').then (_author) =>
      _author.get('contact').then (_contact) =>
        @set 'contactAlias', _contact.get('alias')
  ).property('author.contacts')

  createdAtFromNow: (->
    moment(@get('postedAt')).fromNow()
  ).property('createdAt')

  commentHeader: (->
    @get('profilePictureCall')
    @get('contact')
    if @get('author.id') < 2
      "You commented " + @get('createdAtFromNow')
    else
      @get('contactAlias') + " ( " + @get('author.onion') + " ) commented " + @get('createdAtFromNow')
  ).property('createdAtFromNow', 'author.id', 'author.onion', 'contactAlias')

  commentPending: (->
    @get('isRemotePublished')
  ).property("isRemotePublished")
