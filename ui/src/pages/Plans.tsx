export function Plans() {
  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-3xl font-bold">Planes</h1>
        <button className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg">
          Nuevo Plan
        </button>
      </div>
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Nombre
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Precio
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Intervalo
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Estado
              </th>
              <th className="px-6 py-3"></th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-b hover:bg-gray-50">
              <td colSpan={5} className="px-6 py-8 text-center text-gray-500">
                No hay planes aún
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  )
}
