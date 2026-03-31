import { useKnirvana } from "../../lib/stores/useKnirvana";
import { Card, CardContent } from "./card";

export default function ResourceDisplay() {
  const { nrnBalance, skillsLearned, errorsResolved } = useKnirvana();

  return (
    <Card className="bg-black bg-opacity-80 border-cyan-500">
      <CardContent className="py-3">
        <div className="flex space-x-6 text-sm">
          <div className="text-cyan-300">
            <span className="text-cyan-400 font-mono">NRN:</span>
            <span className="ml-2 text-green-400 font-bold">{nrnBalance.toFixed(1)}</span>
          </div>
          
          <div className="text-cyan-300">
            <span className="text-cyan-400 font-mono">Skills:</span>
            <span className="ml-2 text-yellow-400 font-bold">{skillsLearned}</span>
          </div>
          
          <div className="text-cyan-300">
            <span className="text-cyan-400 font-mono">Resolved:</span>
            <span className="ml-2 text-purple-400 font-bold">{errorsResolved}</span>
          </div>
          
          <div className="text-cyan-300">
            <span className="text-cyan-400 font-mono">Network:</span>
            <span className="ml-2 text-blue-400">D-TEN</span>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
