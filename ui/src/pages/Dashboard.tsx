export function Dashboard() {
  return (
    <div>
      <h1 className="text-3xl font-bold mb-6">Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-gray-600 text-sm font-medium mb-2">
            Suscripciones Activas
          </h3>
          <p className="text-3xl font-bold text-gray-900">0</p>
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-gray-600 text-sm font-medium mb-2">
            Pagos Este Mes
          </h3>
          <p className="text-3xl font-bold text-gray-900">$0.00</p>
        </div>
        <div className="bg-white rounded-lg shadow p-6">
          <h3 className="text-gray-600 text-sm font-medium mb-2">
            Suscripciones Vencidas
          </h3>
          <p className="text-3xl font-bold text-red-600">0</p>
        </div>
      </div>
    </div>
  )
}
