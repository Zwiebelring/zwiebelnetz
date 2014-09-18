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
SecSocNet.ProfilesController = Ember.ArrayController.extend
  allProfiles: null
  content: (->
    @get('allProfiles')
  ).property('allProfiles')
  selectedCircles: []

  # DO NOT DELETE
  myOnion: (->
    @store.find('onion', 1)
  ).property()
  
  picture: (->
    pic = 'img/no_profile_picture.jpg'
    if @get('allProfiles')
      @get('allProfiles').forEach (profile) ->
        if profile.get('key') == 'picture'
          pic = "data:image/jpg;base64," + profile.get('value')
    pic
  ).property('allProfiles.@each')

  selfContact: (->
    @store.find('contact', 1)
  ).property()
  profileKey: ""
  profileValue: ""
  search: ""

  filterProfiles: ->
    if @get('search') == ''
      @set 'content', @get('allProfiles')
    else
      regexp = new RegExp(@get('search'),"i")
      @set 'content', @get('allProfiles').filter (item, index, self) ->
        circleMatched = item.get('circles').any (circ) ->
          circ.get('name').match(regexp)
        circleMatched || item.get('key').match(regexp) || item.get('value').match(regexp)

  searchProfilesObserver: (->
    Ember.run.debounce(@,@filterProfiles,500)
  ).observes('search')

  actions:
    addCircle: (id) ->
      @get('selectedCircles').push(id)
    removeCircle: (id) ->
      idx = @get('selectedCircles').indexOf(id)
      if idx != -1
        @get('selectedCircles').splice(idx, 1);
    createProfile: ->
      key = @get('profileKey')
      value = @get('profileValue')
      return unless key.trim() && value.trim()
      profile = @store.createRecord 'profile',
        key: key
        value: value

      @store.filter('circle', {}, (circle) => parseInt(circle.id) in @get('selectedCircles')).then (_circles) =>
        profile.get('circles').then (_circs) =>
          _circs.pushObjects(_circles)
          profile.save().then( =>
            @set 'profileKey', ""
            @set 'profileValue', ""
            @store.find('onion', 1).then (_myOnion) =>
              _myOnion.get('profiles').then (_profiles) =>
                _profiles.pushObject(profile)
          , (->))
