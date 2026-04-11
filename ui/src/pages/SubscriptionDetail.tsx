import { useParams } from 'react-router-dom'

export function SubscriptionDetail() {
  const { key } = useParams<{ key: string }>()

  return (
    <div>
      <h1 className="text-3xl font-bold mb-6">Detalles de Suscripción</h1>
      <p className="text-gray-600">
        Suscripción: <code className="bg-gray-100 px-2 py-1 rounded">{key}</code>
      </p>

      <div className="mt-6 grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold mb-4">Información General</h2>
          <dl className="space-y-3 text-sm">
            <div>
              <dt className="text-gray-600">Estado</dt>
              <dd className="text-gray-900 font-medium">-</dd>
            </div>
            <div>
              <dt className="text-gray-600">Plan</dt>
              <dd className="text-gray-900 font-medium">-</dd>
            </div>
            <div>
              <dt className="text-gray-600">Monto</dt>
              <dd className="text-gray-900 font-medium">-</dd>
            </div>
          </dl>
        </div>

        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold mb-4">Períodos</h2>
          <dl className="space-y-3 text-sm">
            <div>
              <dt className="text-gray-600">Período Actual</dt>
              <dd className="text-gray-900 font-medium">- a -</dd>
            </div>
            <div>
              <dt className="text-gray-600">Cliente Externo</dt>
              <dd className="text-gray-900 font-medium">-</dd>
            </div>
          </dl>
        </div>
      </div>

      <div className="mt-6 bg-white rounded-lg shadow overflow-hidden">
        <div className="px-6 py-4 border-b bg-gray-50">
          <h2 className="text-lg font-semibold">Últimos Pagos</h2>
        </div>
        <table className="w-full">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Fecha
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Monto
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Estado
              </th>
              <th className="px-6 py-3 text-left text-sm font-medium text-gray-900">
                Proveedor
              </th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-b hover:bg-gray-50">
              <td colSpan={4} className="px-6 py-8 text-center text-gray-500">
                No hay pagos registrados
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  )
}
