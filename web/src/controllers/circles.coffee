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
SecSocNet.CirclesController = Ember.ArrayController.extend
  sortAscending: true
  sortProperties: ["id"]

  circles: Ember.computed.filterBy('allCircles','creator',1)

  contacts: (->
    @get('allContacts')
  ).property('allContacts')	

  allCircles: null
  allContacts: null
  search: ''

  circleCount: (->
    @get('circles').get('length')
  ).property('circles.@each')

  contactCount: (->
    @get('contacts').get('length')-1  # -1 to remove own contact from contactCount
  ).property('contacts.@each')

  filterContacts: ->
    if @get('search') == ''
      @set 'contacts', @get('allContacts')
    else
      regexp = new RegExp(@get('search'))
      @set 'contacts', @get('allContacts').filter (item, index, self) ->
        circleMatched = item.get('circles').any (circ) ->
          circ.get('name').match(regexp)
        circleMatched || item.get('alias').match(regexp)


  searchContactsObserver: (->
    Ember.run.debounce(@,@filterContacts,500)
  ).observes('search')

  actions:
    createCircle: ->
      name = @get('circleName')
      return unless name.trim()
      circle = @store.createRecord 'circle',
        creator: 1
        name: name
      circle.save()
      @set 'circleName', ''
    deleteCircleformContact: (contact, circle) ->
      contact.get('circles').then (_circles) ->
        _circles.removeObject(circle)
      contact.save()
