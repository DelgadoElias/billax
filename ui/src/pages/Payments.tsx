export function Payments() {
  return (
    <div>
      <h1 className="text-3xl font-bold mb-6">Pagos</h1>

      <div className="mb-6 bg-white rounded-lg shadow p-4">
        <div className="flex gap-4 items-end">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Proveedor
            </label>
            <select className="border border-gray-300 rounded-lg px-3 py-2">
              <option value="">Todos</option>
              <option value="mercadopago">Mercado Pago</option>
              <option value="stripe">Stripe</option>
            </select>
          </div>
          <button className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg">
            Filtrar
          </button>
        </div>
      </div>

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Fecha
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Suscripción
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Monto
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Proveedor
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Estado
              </th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-b hover:bg-gray-50">
              <td colSpan={5} className="px-6 py-8 text-center text-gray-500">
                No hay pagos aún
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  )
}
