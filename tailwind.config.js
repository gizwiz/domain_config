/** @type {import('tailwindcss').Config} */
export default {
  //content: ["./views/**/*.templ"], // this is where our templates are located
  content: ["./views/**/*.templ"], // this is where our templates are located
  theme: {
    extend: {
      cursor: {
        busy: 'wait'
      }
    },
  },
  plugins: [],
};
