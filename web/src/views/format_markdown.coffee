marked.setOptions
  renderer: new marked.Renderer()
  gfm: true
  tables: true
  breaks: false
  pedantic: false
  sanitize: true
  smartLists: true
  smartypants: false

Ember.Handlebars.helper "format-markdown", (input) ->
  new Handlebars.SafeString(marked(input))