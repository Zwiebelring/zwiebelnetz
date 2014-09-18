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
SecSocNet.ApplicationController = Ember.Controller.extend
  init: ->
    @_super();
    user  = Ember.$.cookie('auth_user');
    token = Ember.$.cookie('auth_token');
    if (!Ember.isEmpty(user) && !Ember.isEmpty(token))
      @set 'isAuthorized', true
      Ember.$.ajaxPrefilter (options, oriOptions, jqXHR ) ->
        jqXHR.setRequestHeader("Auth-User", user)
        jqXHR.setRequestHeader("Auth-Token", token)
    else
      @set "isAuthorized", false
  hasError: false
  actions:
    ping: ->
      @set 'isAuthorized', true
    logout: ->
      @set 'isAuthorized', false
      @set 'username', ""
      @set 'password', ""
      Ember.$.removeCookie('auth_user');
      Ember.$.removeCookie('auth_token');
      Ember.$.ajaxPrefilter (options, oriOptions, jqXHR ) ->
        jqXHR.setRequestHeader("Auth-User", "")
        jqXHR.setRequestHeader("Auth-Token", "")
    authorize: ->
      user = @get("username")
      pass = @get("password")
      payload = "{\"Username\":\"#{user}\",\"Password\":\"#{pass}\"}"
      context = this
      $.ajax
        type: "POST"
        url: "/api/authorize"
        data: payload
        success: (data, status, nil) ->
          context.set "auth_token", data["auth_token"]
          context.set "isAuthorized", true
          context.set "hasError", false
          Ember.$.cookie 'auth_user', user
          Ember.$.cookie 'auth_token', data["auth_token"]
          Ember.$.ajaxPrefilter (options, oriOptions, jqXHR ) ->
            jqXHR.setRequestHeader("Auth-User", user)
            jqXHR.setRequestHeader("Auth-Token", data["auth_token"])
          location.reload()
        error: (xhr, ajaxOptions, error) ->
          context.set "password", ""
          context.set "hasError", true
        dataType: "json"
