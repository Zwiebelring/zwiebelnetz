SecSocNet.CircleSelectView = Ember.View.extend
  tagName: 'div'
  width: '55%'
  placeholder: "Share with"
  attributeBindings: ['circleInit','selected','width']

  controllerToView: ( ->
    @$().select2 'val', @get('circleInit').mapBy('id')
  ).observes('circleInit.@each')

  didInsertElement: ->
    console.log("select2 is required for SelectOrder") unless @$().select2

    @$().select2
      multiple: true
      placeholder: @get('placeholder')
      width: @get('width')
      #data: @get('value')
      ajax:
        headers:
          "Auth-User": $.cookie 'auth_user'
          "Auth-Token": $.cookie 'auth_token'
        url: "/api/circles"
        async: false
        results: (data, page, query) ->
          regexp = new RegExp(query.term)
          filteredResults = data.circles.filter (item) -> item.name.match(regexp)
          categories = []
          children = filteredResults.filter (item) -> item.creator == 1 || item.name == "Public"
          if children.length > 0
            categories.push( text: 'Circles', children: children)
          children = filteredResults.filter (item) -> item.creator == 0 && item.name != "Public"
          if children.length > 0
            categories.push( text: 'Contacts', children: children)
          return results: categories

      formatResult: format
      formatSelection: format
      dropdownCssClass: "bigdrop"
      tokenSeparators: [",", " "]
      initSelection: (element, callback) ->
        ids = element.val().split(",")
        uri = "/api/circles?"
        id = ids.pop()
        return unless id.trim()
        uri += ("ids%5B%5D=" + id)
        uri += ("&ids%5B%5D=" + id) for id in ids
        $.ajax(uri,
          headers:
            "Auth-User": $.cookie 'auth_user'
            "Auth-Token": $.cookie 'auth_token'
        ).done (data) ->
          callback(data.circles)

    if @get('circleInit') && @get('circleInit').isFulfilled
      @$().select2 'val', @get('circleInit').mapBy('id')

    @$().on "change", (event) =>
      if event.added
        @get('controller').send('addCircle', event.added.id)
      if event.removed
        @get('controller').send('removeCircle', event.removed.id)

formatWrapper = (object, container, query) ->
  format(object)

format = (object) ->
  return object.text if object.text
  if object.creator == 0 && object.name != "Public"
    "<span class=\"glyphicon glyphicon-user\"></span> " + object.name
  else
    "<span class=\"glyphicon\"><img src='img/group.png'></span> " + object.name

