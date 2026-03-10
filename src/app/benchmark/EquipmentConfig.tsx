"use client";
import React, { useState } from "react";

export default function EquipmentConfig({ onChange }) {
  const [numEquipments, setNumEquipments] = useState(1);

  const handleChange = (e) => {
    const value = parseInt(e.target.value, 10);
    setNumEquipments(value);
    onChange(value);
  };

  return (
    <div className="mb-8">
      <label className="font-medium mr-2">Number of Equipments:</label>
      <input
        type="number"
        min={1}
        value={numEquipments}
        onChange={handleChange}
        className="border px-2 py-1 rounded w-24"
      />
    </div>
  );
}
