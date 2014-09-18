#  -*-  indent-tabs-mode: nil; c-basic-offset: 4; tab-width: 4 -*- 
SecSocNet.CircleDrag = Ember.View.extend
  templateName: 'circles-drag'
  attributeBindings: ['draggable']
  draggable: "true"

  dragStart: (ev) ->
    ev.dataTransfer.setData('text/data', @get('content.id'))
