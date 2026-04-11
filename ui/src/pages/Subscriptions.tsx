export function Subscriptions() {
  return (
    <div>
      <h1 className="text-3xl font-bold mb-6">Suscripciones</h1>
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Clave
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Cliente
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Estado
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Plan
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Monto
              </th>
              <th className="px-6 py-3"></th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-b hover:bg-gray-50">
              <td colSpan={6} className="px-6 py-8 text-center text-gray-500">
                No hay suscripciones aún
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  )
}
