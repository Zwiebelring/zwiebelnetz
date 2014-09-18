#  -*-  indent-tabs-mode: nil; c-basic-offset: 4; tab-width: 4 -*- 
SecSocNet.ProfileCircleDrop = Ember.View.extend
  templateName: 'profile-circles-drop'
  dragOver: (ev) ->
    ev.preventDefault()
  drop: (ev) ->
    id = ev.dataTransfer.getData('text/data')
    controller = @get('controller')#.get().findProperty('id', Number(id))
    circle = controller.get('circles').filterBy('id',id)[0]
    profile = controller.get('profiles').filterBy('id',@get('content.id'))[0]

    profile.get('circles').then (_circles) ->
      _circles.pushObject(circle)
      profile.save().then( (->),-> profile.reload())
    #circle.get('contacts').then(c) =>
    #  c.pushObject(circle)
    #  c.save()
    #contact.get('circles').then(c) =>
    #  c.pushObject(contact)
    #  c.save()
