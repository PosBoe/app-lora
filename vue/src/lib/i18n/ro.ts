// The Romanian messages. They are the schema: ru.ts must hold the same keys.
export const ro = {
  app: {
    name: 'lora',
    language: 'Limba',
    logout: 'Ieșire',
  },
  nav: {
    organizations: 'Organizații',
    devices: 'Dispozitivele mele',
  },
  common: {
    cancel: 'Anulează',
    close: 'Închide',
    confirm: 'Confirmă',
    copy: 'Copiază',
    copied: 'Copiat',
    retry: 'Reîncearcă',
  },
  login: {
    title: 'Autentificare',
    login: 'Login',
    password: 'Parolă',
    submit: 'Intră',
  },
  setPassword: {
    title: 'Setează parola ta',
    intro:
      'Parola primită de la administrator este temporară. Alege o parolă nouă pentru a continua.',
    current: 'Parola temporară',
    next: 'Parola nouă',
    repeat: 'Repetă parola nouă',
    mismatch: 'Parolele nu coincid.',
    rule: 'Cel puțin 8 caractere.',
    submit: 'Salvează parola',
  },
  home: {
    title: 'Dispozitivele mele',
    empty: 'Aici vor apărea dispozitivele organizației tale.',
  },
  organizations: {
    title: 'Organizații',
    add: 'Adaugă organizație',
    empty: 'Nu există organizații.',
    name: 'Nume',
    status: 'Stare',
    active: 'Activă',
    deactivated: 'Dezactivată',
    login: 'Login',
    profile: 'Profil',
    profileMissing: 'Profil necompletat',
    lastLogin: 'Ultima logare',
    never: 'Niciodată',
    created: 'Creată',
    actions: 'Acțiuni',
    resetPassword: 'Resetează parola',
    deactivate: 'Dezactivează',
    reactivate: 'Reactivează',
    createTitle: 'Organizație nouă',
    createSubmit: 'Creează',
    loginHint: '3–64 caractere: a-z, 0-9, punct, cratimă, underscore.',
    resetConfirm:
      'Utilizatorul {login} va primi o parolă temporară nouă și va fi deconectat. Continui?',
    deactivateConfirm:
      'Utilizatorii organizației {name} nu se vor mai putea loga. Datele și integrările rămân. Continui?',
    reactivateConfirm: 'Utilizatorii organizației {name} se vor putea loga din nou. Continui?',
  },
  temporaryPassword: {
    title: 'Parolă temporară',
    intro: 'Trimite aceste date utilizatorului. Parola se afișează o singură dată.',
    login: 'Login',
    password: 'Parolă',
  },
  errors: {
    invalid_credentials: 'Loginul sau parola este greșită.',
    login_locked: 'Prea multe încercări greșite. Încearcă din nou peste câteva minute.',
    organization_deactivated: 'Accesul este suspendat. Contactați administratorul.',
    unauthenticated: 'Sesiunea a expirat. Autentifică-te din nou.',
    password_change_required: 'Setează mai întâi o parolă nouă.',
    forbidden: 'Nu ai acces la această acțiune.',
    invalid_current_password: 'Parola curentă este greșită.',
    password_too_short: 'Parola trebuie să aibă cel puțin 8 caractere.',
    password_too_long: 'Parola este prea lungă.',
    login_invalid: 'Loginul poate avea 3–64 caractere: a-z, 0-9, punct, cratimă, underscore.',
    login_taken: 'Acest login este deja folosit.',
    name_required: 'Numele organizației este obligatoriu.',
    not_found: 'Nu a fost găsit.',
    unknown: 'A apărut o eroare. Încearcă din nou.',
  },
  notFound: {
    title: 'Pagina nu există',
    text: 'Pagina pe care ai deschis-o nu există.',
    home: 'Mergi la pagina principală',
  },
}

// Messages is the shape every language must have.
export type Messages = typeof ro
