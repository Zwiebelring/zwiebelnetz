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

SecSocNet.Contact = DS.Model.extend
  onion:            DS.belongsTo('onion', async: true)
  circles:          DS.hasMany('circle', async: true)
  nickname:         DS.attr('string')
  alias:            DS.attr('string')
  trust:            DS.attr('number')
  status:           DS.attr('number')
  request_message:  DS.attr('string')

  isSelfContact: (->
    @get("id") <= "1" # sometimes, there is a broken id==0 contact...
  ).property("id")

  isSelectedContacts: (->
    console.log("spam")
    contact = @get('model')
    console.log(contact.id)
    console.log(selectedContact.id)
    contact.id == selectedContact.id
  )

  userCreatedCircles: Ember.computed.filterBy('circles','creator',1)

  isBlocked: (->
    @get("status") == 0
  ).property("status")
  isOpen: (->
    @get("status") == 1
  ).property("status")
  isPending: (->
    @get("status") == 2
  ).property("status")
  isSuccess: (->
    @get("status") == 3
  ).property("status")
  isFollowing: (->
    @get("status") == 4
  ).property("status")

  profilePictureId: DS.attr 'number'
  profilePicture: "img/no_profile_picture.jpg"
  profilePictureCall: (->
    if @get('profilePictureId') > 0              
      profile = @store.find('profile', @get('profilePictureId')).then (_profile) =>          
        @set 'profilePicture',  "data:image/jpg;base64," + _profile.get('value')
    else
      @set 'profilePicture', "img/no_profile_picture.jpg"
    ""
  ).property('profilePictureId')
