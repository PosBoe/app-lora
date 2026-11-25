// What a component accepts that its props do not name. Vue hands an attribute
// it does not recognise to the element the component renders, so `@click` on
// `<v-btn>` reaches the button. The template checker (strictTemplates) checks
// only the declared props, and Vuetify 4's VBtn declares no onClick.
//
// Vue merges ComponentCustomProps into the props type of every component, so a
// name declared here is accepted on any component. The price is that the
// checker never checks that name again. Add a name only when a template needs
// it, and only a native DOM event or a native attribute of the rendered element.

export {}

declare module 'vue' {
  interface ComponentCustomProps {
    // Native DOM events.
    onClick?: (payload: MouseEvent) => void
    onKeydown?: (payload: KeyboardEvent) => void
    onFocus?: (payload: FocusEvent) => void
    onBlur?: (payload: FocusEvent) => void

    // Native attributes of the rendered element.
    title?: string
    type?: string
    // The file types that v-file-input passes to its native input.
    accept?: string
  }
}
