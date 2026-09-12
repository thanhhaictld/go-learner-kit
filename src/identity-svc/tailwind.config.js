module.exports = {
  content: ["./Areas/Identity/Pages/**/*.cshtml", "./Views/**/*.cshtml"],
  darkMode: "media",
  theme: { extend: {} },
  plugins: [require("@tailwindcss/forms"), require("@tailwindcss/typography")]
};
