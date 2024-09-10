import React, { useState, useRef } from "react";

export default function Component() {
  const [inputText, setInputText] = useState("");
  const [outputText, setOutputText] = useState("");
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (e) => {
        const content = e.target?.result as string;
        setInputText(content);
      };
      reader.readAsText(file);
    }
  };

  const handleExecute = async () => {
    try {
      const response = await fetch("http://localhost:3000/analyze", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ command: inputText }),
      });

      if (!response.ok) {
        throw new Error("Network response was not ok");
      }

      const data = await response.json();
      const results = data.results.join("\n");
      setOutputText(results);
    } catch (error) {
      console.error("Error:", error);
      setOutputText(`Error: ${error}`);
    }
  };

  return (
    <div className="flex flex-col min-h-screen bg-gradient-to-br from-purple-500 to-pink-400">
      <div className="flex-grow flex items-center justify-center p-4">
        <div className="w-full max-w-8xl p-6 bg-white rounded-lg shadow-lg">
          <h1 className="text-2xl font-semibold text-center mb-4 text-gray-800">
            Componente de Análisis de Texto
          </h1>
          <div className="flex justify-between mb-4">
            <input
              type="file"
              ref={fileInputRef}
              onChange={handleFileChange}
              className="hidden"
              accept=".txt"
            />
            <button
              onClick={() => fileInputRef.current?.click()}
              className="px-5 py-2 bg-indigo-500 text-white font-medium rounded-md hover:bg-indigo-600 transition-colors duration-200 shadow-md focus:outline-none focus:ring-2 focus:ring-indigo-400"
            >
              Examinar Archivo
            </button>
            <button
              onClick={handleExecute}
              className="px-5 py-2 bg-teal-500 text-white font-medium rounded-md hover:bg-teal-600 transition-colors duration-200 shadow-md focus:outline-none focus:ring-2 focus:ring-teal-400"
            >
              Ejecutar Análisis
            </button>
          </div>
          <div className="flex space-x-4">
            <textarea
              className="w-3/5 h-64 p-3 border border-purple-300 rounded-md resize-none focus:outline-none focus:ring-2 focus:ring-purple-500 shadow-sm"
              value={inputText}
              onChange={(e) => setInputText(e.target.value)}
              placeholder="Ingrese el texto aquí..."
            />
            <textarea
              className="w-3/5 h-64 p-3 border border-purple-300 rounded-md resize-none bg-purple-100 focus:outline-none shadow-inner"
              value={outputText}
              readOnly
              placeholder="Resultados se mostrarán aquí..."
            />
          </div>
        </div>
      </div>
      <footer className="py-4 text-center text-sm text-gray-700 bg-white">
        © {new Date().getFullYear()} Juan Jose Almengor Tizol 202212209..
      </footer>
    </div>
  );
}
