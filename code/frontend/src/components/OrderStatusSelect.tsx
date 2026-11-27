import OrderField from "./OrderField";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "./ui/select";

/**
 * The status control: the status the order has and the moves the shop allows
 * from it. It is built on the kit's Select rather than on Choice, because
 * Choice forwards no `aria-invalid` or `aria-describedby`, and a refused move
 * ("An order cannot go from cancelled to packed.") has to be marked on the
 * control and announced beside it.
 */
export default function OrderStatusSelect({ label, value, options, error, onChange }: {
  label: string;
  value: string;
  options: readonly { value: string; label: string }[];
  error?: string;
  onChange: (value: string) => void;
}) {
  return (
    <OrderField label={label} error={error}>
      {(described) => (
        <Select value={value} onValueChange={onChange}>
          <SelectTrigger aria-label={label} className="w-full bg-surface font-normal" {...described}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {options.map((o) => <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>)}
          </SelectContent>
        </Select>
      )}
    </OrderField>
  );
}
