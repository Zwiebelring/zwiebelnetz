#  -*-  indent-tabs-mode: nil; c-basic-offset: 4; tab-width: 4 -*- 
SecSocNet.PictureDrop = Ember.View.extend
  templateName: 'picture-drop'
  attributeBindings: ['currentImage']

  imgWidth: 100
  imgHeight: 100
  imgSource: ""
  isUploading: false
  hasProfilePicture: false

  imgSource: (->
    if @get('currentImage').indexOf("img",0) == 0
      @set 'hasProfilePicture', false
    else
      @set 'hasProfilePicture', true
    @get('currentImage')
  ).property('currentImage')

  actions:
    deleteProfilePicture: ->
      Ember.$.ajax
        type: "DELETE"
        url: "/api/profile_picture"
        success: (data, status, nil) =>
          @set 'imgSource', 'img/no_profile_picture.jpg'
          @set 'hasProfilePicture', false
        error: (xhr, ajaxOptions, error) =>
          @set 'imgSource', 'img/no_profile_picture.jpg'
          @set 'hasProfilePicture', false

  didInsertElement: ->

    $('#f').change (ev) =>
      file = ev.target.files[0]
      @uploadPicture(file)

    @$('.picture-zone').on "click", (event) =>
      $('#f').click()

  dragOver: (ev) ->
    ev.preventDefault()
  drop: (ev) ->
    ev.preventDefault()
    file = ev.dataTransfer.files[0]
    @uploadPicture(file)

  uploadPicture: (file) ->
    that = @
    if typeof(FileReader) != "undefined" && (/image/i).test(file.type)
      reader = new FileReader()
      reader.onload = (event) ->
        Ember.run ->
          that.set 'imgSource', event.target.result
          #that.set 'isUploading', true
      reader.readAsDataURL(file)

    user  = Ember.$.cookie('auth_user')
    token = Ember.$.cookie('auth_token')
    that.$('#uploadProgress').show()
    xhr = new XMLHttpRequest()
    xhr.upload.addEventListener("progress", (evt) ->
      if evt.lengthComputable
        Ember.run ->
          $('#uploadBar')[0].style.width = (evt.loaded / evt.total) * 100 + "%"
    , false)
    xhr.addEventListener("load", ->
      Ember.run ->
        #that.set 'isUploading', false
        that.$('#uploadProgress').hide()
        that.set 'hasProfilePicture', true
    , false)

    xhr.open("post", "/profile_picture", true);
    xhr.setRequestHeader('Auth-User', user)
    xhr.setRequestHeader('Auth-Token', token)
    formData = new FormData();
    formData.append('file', file)
    formData.append('title', file.name)
    xhr.send(formData);
